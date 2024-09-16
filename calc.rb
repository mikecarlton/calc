#!/usr/bin/env ruby

begin
require 'sqlite3'
rescue LoadError => e
end

=begin
MIT License

Copyright (c) 2017-2023 Mike Carlton

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
=end

require 'bigdecimal'
require 'forwardable'
require 'fileutils'
require 'date'
require 'json'
require 'net/http'
require 'open3'
require 'strscan'

def red(msg)
  "\e[31m#{msg}\e[0m"
end

def green(msg)
  "\e[32m#{msg}\e[0m"
end

def die(*msgs)
  warn(red(*msgs))
  exit(1)
end

version = RUBY_VERSION.split(".").map(&:to_i)
die red("This code requires Ruby 2.7 or greater") unless (version[0] > 2 || version[0] == 2 && version[1] >= 7)

$options = {
  verbose: 0,
  precision: 2,
}

OPTIONS = [
  [ "-t", nil,     "Trace operations", ->(opts) { opts[:trace] = true } ],
  [ "-b", nil,     "Show binary representation of integers", ->(opts) { opts[:binary] = true } ],
  [ "-x", nil,     "Show hex representation of integers", ->(opts) { opts[:hex] = true } ],
  [ "-i", nil,     "Show IPv4 representation of integers", ->(opts) { opts[:ipv4] = true } ],
  [ "-a", nil,     "Show ASCII representation of integers", ->(opts) { opts[:ascii] = true } ],
  [ "-c", Integer, "Column to extract from lines on stdin (negative counts from end)",
                                       ->(opts, val) { raise ArgumentError.new("Column cannot be 0") if val == 0
                                                       opts[:column] = val } ],
  [ "-d", Regexp,  "Regular expression to split columns (default: whitespace)", ->(opts, val) { opts[:delimiter] = val } ],
  [ "-p", Integer, "Set display precision for floating point number (default: #{$options[:precision]})",
                                       ->(opts, val) { raise ArgumentError.new("Precision cannot be negative") if val < 0
                                                        opts[:precision] = val } ],
  [ "-g", nil,     "Use ',' to group decimal numbers", ->(opts) { opts[:grouped] = ',' } ],
  [ "-f", nil,     "Show prime factorization of integers", ->(opts) { opts[:factor] = true } ],
  [ "-s", nil,     "Show statistics of values", ->(opts) { opts[:stats] = true } ],
  [ "-q", nil,     "Do not show stack at finish", ->(opts) { opts[:quiet] = true } ],
  [ "-o", nil,     "Show final stack on one line", ->(opts) { opts[:oneline] = true } ],
  [ "-D", Date,    "Date for currency conversion rates (e.g. 2022-01-01)", ->(opts, val) { opts[:date] = val.strftime('%F') } ],
  [ "-v", nil,     "Verbose output (repeat for additional output)", ->(opts) { opts[:verbose] += 1 } ],
  [ "-u", nil,     "Show units", ->(opts) { units ; exit } ],
  [ "-h", nil,     "Show extended help", ->(opts) { help ; exit } ],
]

def usage
  warn "Usage: [ARGUMENTS | ] #{File.basename($0)} [OPTIONS | ARGUMENTS] [--] ARGUMENT(S)"
  warn
  warn "Options:"

  OPTIONS.each do |option|
    warn "%3s %-7s %s" % option[0,3]
  end
end

def help
  usage

  puts <<~EOS

  Input is read from stdin and then the command line.

  Numbers (with optional leading '-'; can use ',' or '_' to group digits):
      Integers: decimal, binary (leading 0b), or hexadecimal (leading 0x)
      Rationals: <integer>/<integer>
      Floats: decimal, with optional exponent (E)

      Integers can have a final binary magnitude factor (KMGTPEZY) for
      kilo-, mega-, giga-, tera-, peta-, exa-, zetta- or yotta-byte

  Arithmetic operations (prepend '@' to reduce the stack):
      + - * • . / ÷ % ** pow (aliases: • and . for *, ÷ for /, pow for **)
      lcm, gcd (integers only)

  Bitwise operations (integers only, prepend '@' to reduce the stack):
      & | ^

  Bitwise shift operations (integers only):
      << >>

  Unary operations:
              ~: bitwise complement (integer only)
              !: factorial (integer only)
     t truncate: truncate to integer
          round: round to integer
              [: floor
              ]: ceiling
              i: invert units
              r: reciprocal (1/x)
            chs: change sign
      √ sq sqrt: sqrt

  Math functions:
            rand: push random number in range [0..1)
            log, log2, log10, sin, cos, tan

  Stack manipulation:
          x: exchange top two elements
      p pop: pop topmost element
      d dup: duplicate topmost element
      clear: clear entire stack

  Stack operations (pushes the new value, append '!' to replace stack):
      mean max min size

  Constants:
      pi π
      inf infinity ∞
      e
      C

  Registers (displayed at exit):
      >NAME: pop topmost element and save in NAME
      <NAME: push NAME onto stack
      >:NAME: clear NAME

  Special Characters:
      π option-p
      ∞ option-5
      • option-8
      ÷ option-/
      √ option-v

   ASCII:
      Surround with single quotes, i.e. 'fubar', will be converted to an integer

   Stock Quote:
      Proceed with @, i.e. @aapl, will be converted to most recent quote in USD
      With -v option also prints extended quote info

   Units:
      Units are applied if current top of stack does not have a numerator or denominator
      Otherwise the current top of stack is converted to the units

  EOS
  units(indent: 4)
end

def units(indent: 0)
  dimensions = Unit.all.group_by(&:dimension)
  dimensions.each do |dimension, units|
    next unless dimension

    puts "#{' '*indent}#{dimension}"
    units.group_by(&:si).each do |_si, set|
      puts ' '*(indent+2) + set.map { |u| "#{u.desc} (#{u.name})" }.join(', ')
    end
  end
end

def get_url(url, token: nil, params: nil)
  warn "[get(#{url})]" if $options[:trace]
  uri = URI(url)
  uri += '?' + URI.encode_www_form(params) if params
  headers = { }
  headers['Authorization'] = "Token #{token}" if token

  response = Net::HTTP.get_response(uri, headers)
  if response.is_a?(Net::HTTPSuccess)
    type = response.header['content-type']&.split(';').first.downcase
    case type
    when "application/json"
      JSON.parse(response.body).tap { |resp| warn(JSON.pretty_generate(resp)) if $options[:verbose] > 1 }
    else
      warn(red("Unknown response type '#{type}' from '#{url}'"))
      nil
    end
  else
    warn(red("HTTP failure '#{response.code}' from '#{url}'"))
    nil
  end
end

def get_api_key(source)
  if ENV[source]
    warn "[ENV['#{source}']]" if $options[:trace]
    api_key = ENV[source]
  end

  if !api_key && RUBY_PLATFORM =~ /darwin/
    warn "[security(#{source})]" if $options[:trace]
    # handle remote access where keychain might be locked:
    # security show-keychain-info || security unlock-keychain
    security_cmd = %W(security find-generic-password -s #{source} -a api_key -w)
    stdout, _stderr, status = Open3.capture3(*security_cmd)
    api_key = stdout.chomp if status == 0
  end

  die <<~EOS unless api_key
    Please set api_key in security (macos) or the environment, e.g.
      export ENV['#{source}']=$api_key
    or
      security add-generic-password -s #{source} -a api_key -U -w $api_key
  EOS

  api_key
end

def print_table(table, file=$stdout)
  widths = Array.new(table[0].length+1, 0)
  widths = table.inject(widths) do |current, line|
    line.map { |v| v.to_s.gsub(/\e[^m]+m/,'').length }. # map each value to its width without escapes
         zip(current).                # zip with current widths
         map { |w| w.max }            # return max of previous, current width
  end
  widths[0] *= -1 # left justify
  widths[-1] *= -1
  widths[-2] *= -1 if $options[:verbose] > 1

  table.each do |line|
    # need to allow extra width to account for non-printing terminal escapes
    adjusted = widths.zip(line).map { |w, val| [ w+(val.length-val.gsub(/\e[^m]+m/,'').length), val ] }
    file.puts ("%*s  "*line.size) % adjusted.flatten
  end
end

class IPv4Error < ArgumentError
end

class UnitsError < TypeError
end

class Stock
  QUOTES_CACHE = "#{ENV['HOME']}/data/stock.sqlite3"
  @@db = nil

  def open(db_name = QUOTES_CACHE)
    db = SQLite3::Database.new db_name

    # Define a schema.  Values in mills, except market cap
    schema = <<-SQL
      CREATE TABLE IF NOT EXISTS quote (
        id         INTEGER  PRIMARY KEY,
        date       DATE     NOT NULL,
        currency   TEXT     DEFAULT '$',
        symbol     TEXT     NOT NULL,
        open       INTEGER,
        high       INTEGER,
        low        INTEGER,
        close      INTEGER,
        volume     INTEGER,
        marketcap  INTEGER,
        peRatio    INTEGER,
        updated_at DATETIME DEFAULT (datetime('now', 'utc'))
      );
    SQL

    db.execute_batch(schema)
    db
  end

  def get(symbol, date: nil)
  end
end

class String
  # parse a string (with possible ',' and/or '_' separators) into integer
  def int
    /([_,])/.match(self) { |m| $options[:grouped] ||= m[1] }

    # we remove leading 0's so won't be read as octal (PDP-10 is long gone)
    str = self.sub(/\A0+(?=\d)/, '').gsub(/[,_]/, '')

    factor = 1
    final = str[-1]
    if Stack::MAGNITUDE.include? final
      str = str.slice(0..-2)
      factor = 2 ** ((Stack::MAGNITUDE.index(final)+1) * 10)
    end

    Integer(str) * factor
  end

  # parse a IPv4-formatted string into integer
  def ipv4
    input = 0
    Stack::IPV4.match(self) do |m|
      m[1..4].each do |octet|
        octet = Integer(octet)
        raise IPv4Error if octet > 255
        input = input*256 + octet
      end
    end

    input
  end

  def quote_alphavantage
    # see https://www.alphavantage.co/documentation/#time-series-data
    # intraday is adjusted, after close
    # optional month=2024-04
    api_key = get_api_key('alphavantage')
    quote_url = "https://www.alphavantage.co/query"
    params = {
      function: 'TIME_SERIES_INTRADAY',
      symbol: self,
      interval: '60min',        # 1min, 5min, 15min, 30min, 60min
      outputsize: 'compact',    # compact => 100 datapoints, full gets 30 days or entire month
      extended_hours: 'true',
      apikey: api_key,
    }

    response = get_url(quote_url, params: params)
    die "Unable to get quote" unless response

    #  { "Meta Data" => { },
    #    "Time Series (60min)"=>{ "2024-05-31 19:00:00"=>{"1. open"=>"211.3700", "2. high"=>"212.2000", "3. low"=>"211.3000", "4. close"=>"211.4000", "5. volume"=>"148"} ,... } }
    key = "Time Series (#{params[:interval]})"
    most_recent = response[key].first[1]["4. close"]
    value = BigDecimal(most_recent, Float::DIG)
    Denominated(value, :usd)
  end

  def quote_iex
    api_key = get_api_key('iex')
    # see also https://cloud.iexapis.com/stable/stock/#{self}/intraday-prices?sort=desc&token=#{api_key}
    quote_url = "https://api.iex.cloud/v1/data/core/quote"
    response = get_url("#{quote_url}/#{self}?token=#{api_key}")
    die "Unable to get quote" unless response

    if response[0] && response[0]['latestPrice'] && response[0]['currency']
      q = response[0]
      if $options[:verbose] > 0
        def fmt(val, cur=nil, unit: nil, delta:false)
          if val.is_a? Numeric
            neg = val < 0
            val = sprintf(delta ? "%+.02f" : "%.02f", val)
            val = neg ? red(val) : green(val) if delta
          end
          val.to_s + unit.to_s    # cleaner if we don't show the currency
        end

        cur = '$' if q['currency'] == 'USD'
        quote = [ fmt(q['symbol']),
                  fmt(q['latestPrice'], cur),
                  fmt(q['change'], cur, delta:true),
                  fmt(q['changePercent']*100, unit:'%', delta:true),
                  fmt(q['low'], cur),
                  fmt(q['high'], cur),
                  fmt(q['week52Low'], cur),
                  fmt(q['week52High'], cur),
                  fmt(q['ytdChange']*100, unit:'%', delta:true),
                  fmt(q['marketCap']/1e9, cur, unit: 'B'),
                  fmt(q['peRatio']),
                  fmt(Time.at(q['latestUpdate']/1000).strftime('%F %r')) ]
        quote << q['companyName'] if $options[:verbose] > 1

        open = q['isUSMarketOpen'] == 'true'
        Stack.quotes(cur, open) << quote
      end

      value = BigDecimal(response[0]['latestPrice'], Float::DIG)
      currency = response[0]['currency']&.downcase.to_sym
      Denominated(value, currency)
    else
      nil
    end
  end

  def quote
    value = quote_alphavantage
    $stderr.puts("ticker symbol '#{self}' not found") unless value

    value
  end

  # parse a string (with possible ',' and/or '_' separators) into float
  def float
    /([_,])/.match(self) { |m| $options[:grouped] ||= m[1] }

    BigDecimal(self.gsub(/[,_]/, ''), Float::DIG)
  end

  # return distance from end of last occurence of string or regexp
  # returns 0 if not found
  def from_end(str)
    i = rindex(str)
    i ? length - i : 0
  end
end

class BigDecimal
  orig_to_s = instance_method :to_s

  # default to printing floating point format instead of exponential
  define_method :to_s do |*param|
    if self == (Integer(self) rescue nil)     # need rescue to handle Infinity or NaN
      Integer(self).to_s
    else
      orig_to_s.bind(self.round($options[:precision])).call(param.first || 'F')
    end
  end
end

# extend Numeric with methods for Integers, Rationals and Floats
class Numeric
  def format(form)
    form == 10 ? self.grouped($options[:grouped]) : ''
  end

  def grouped(sep = nil)
    s = self.to_s
    s = s.reverse.gsub(/(\d{3})(?=\d+-?$)/, "\\1#{sep}").reverse if sep
    s
  end

  def sqrt
    raise RangeError if self < 0
    self**0.5
  end

  def simplify
    self
  end

  def from(numerator, denominator)
    # currency is special -- can only convert to/from USD
    if numerator&.dimension == :currency && numerator != Unit[:usd] && denominator != Unit[:usd]
      value = numerator.factor.(self)
      numerator = Unit[:usd]
    else
      value = self
    end

    # don't use @numerator.factor/unit.factor -- lose exact integers
    value = numerator.factor.is_a?(Proc) ? numerator.factor.(value) : value*numerator.factor if numerator
    value = denominator.ifactor.is_a?(Proc) ? denominator.ifactor.(value) : value/denominator.factor if denominator
    value
  end

  def to(numerator, denominator)
    value = self
    value /= numerator.factor if numerator
    value *= denominator.factor if denominator
    value
  end

  RATES_CACHE = "#{ENV['HOME']}/data/currency"
  def rates_cache
    ::FileUtils.mkdir_p(RATES_CACHE)
    $options[:date] ? "#{RATES_CACHE}/#{$options[:date]}-rates.json" : "#{RATES_CACHE}/rates.json"
  end

  def rates_url
    $options[:date] ? "https://openexchangerates.org/api/historical/#{$options[:date]}.json"
                    : "https://openexchangerates.org/api/latest.json"
  end

  $rates = nil
  def convert_currency(from, to)
    if !$rates
      if File.exist?(rates_cache)
        warn "[load(#{rates_cache})]" if $options[:trace]
        $rates = JSON.parse(File.read(rates_cache))
      end

      if !$rates || ($options[:date].nil? && $rates['timestamp'] < Time.now.to_i-3600)
        api_key = get_api_key('openexchangerates')
        url = rates_url
        $rates = get_url(url, token: api_key)
        die "Unable to get exchange rates" unless $rates
        File.write(rates_cache, JSON.pretty_generate($rates))
      end
    end

    from = from.to_s.upcase
    to = to.to_s.upcase
    unless $rates['rates'][to] && $rates['rates'][from]
      die "Unable to find exchange rates for #{to if !$rates['rates'][to]} #{from if !$rates['rates'][from]}"
    end

    if from == $rates['base']
      self / BigDecimal($rates['rates'][to], Float::DIG)
    elsif to == $rates['base']
      self * BigDecimal($rates['rates'][from], Float::DIG)
    else
      die "Invalid usage: convert #{from} -> #{to}"
    end
  end
end

class Rational
  def simplify
    self.to_i == self ? self.to_i : self
  end
end

class Integer
  def finite?
    true
  end

  def format(form)
    case form
    when      10 then super
    when      16 then "0x%x" % self
    when       2 then "0b%b" % self
    when   :ipv4 then self.to_ipv4
    when  :ascii then self.to_ascii
    when :factor then self.to_factor
    end
  end

  def factorial
    raise RangeError if self < 0
    (1..self).inject(1) { |r, i| r*i }
  end

  def factors
    raise RangeError if self < 0

    num = self
    limit = Integer(num.sqrt)
    factors = []
    f = 2
    while f <= limit && f < num
      while (num > 1 && num % f == 0)
        factors << f
        num /= f
      end
      f += (f == 2) ? 1 : 2
    end
    factors << num if num > 1

    factors
  end

  def to_ascii
    return '' if self < 0

    v = self
    ary = []
    while v > 0
      v, ch = *(v.divmod(256))
      ary.push(ch >= 32 && ch < 128 ? ch.chr : "\\x%x%x" % ch.divmod(16))
    end

    ary.reverse.join
  end

  def to_ipv4
    return '' if self < 0 || self >= 1<<32

    v = self
    ary = []
    4.times do
      v, byte = *(v.divmod(256))
      ary.push(byte)
    end

    ary.reverse.join('.')
  end

  def to_factor
    return '' if self < 0

    primes = { }
    self.factors.each do |f|
      primes[f] ||= 0
      primes[f] += 1
    end

    primes.map { |p, c| c == 1 ? "#{p}" : "#{p}^#{c}" }.join(' * ')
  end
end

# Define / to do integer division if exact, else floating (e.g. 5/2 => 2.5, 4/2 => 2)
# N.B. Only do this if you know all code expects this behavior
Integer.class_eval do
  current_verbosity = $VERBOSE
  $VERBOSE = false                      # avoid warning about discarding old :/
  original_div = instance_method(:/)
  define_method(:/) do |other|
    begin
      quotient = original_div.bind(self).call(other)
    rescue ZeroDivisionError
      quotient = self == 0 ? Float::NAN : Float::INFINITY
    end
    # promote other to Float unless integer division is exact
    if other.integer? && self != quotient * other
      original_div.bind(self).call(BigDecimal(other))
    else
      quotient
    end
  end
  $VERBOSE = current_verbosity
end

class Array
  def mean
    sum / length            # our :/ will promote to float if not exact
  end

  def sample_variance
    m = mean
    sum2 = self.inject(0) { |accum, i| accum + (i-m)**2 }
    sum2 / (length - 1).to_f
  end

  def standard_deviation
    Math.sqrt(sample_variance)
  end

  def percentile(n)
    sort[(length * (n/100.0)).ceil-1]
  end
end

class Unit
  attr_reader :name, :desc, :dimension, :si, :factor, :ifactor

  @@instances = { }

  def to_s
    @name.to_s
  end

  def self.all
    @@instances.values
  end

  def self.names
    @@instances.keys
  end

  def self.[](key)
    @@instances[key.to_sym]
  end

  # factor is amount to multiply to get the base unit
  def initialize(name, desc:, dimension:, si: nil, factor: nil, ifactor: nil)
    @name = name.to_sym
    @desc = desc
    @dimension = dimension
    @si = si
    @factor = factor.is_a?(Float) ? BigDecimal(factor, Float::DIG) : factor
    @ifactor = ifactor.is_a?(Float) ? BigDecimal(ifactor, Float::DIG) : ifactor
    freeze
    @@instances[name] = self
  end

  def commensurable?(other)
    @dimension == other.dimension
  end
end

# order matters: longest prefix match first
Unit.new( :s, desc: 'seconds', dimension: :time, factor: 1)
Unit.new(:mn, desc: 'minutes', dimension: :time, factor: 60)
Unit.new(:hr, desc: 'hours',   dimension: :time, factor: 3600)

# base unit for length is meters
Unit.new(:mm, desc: 'millimeters', dimension: :length, si: true,  factor: 1/1000)
Unit.new(:cm, desc: 'centimeters', dimension: :length, si: true,  factor: 1/100)
Unit.new( :m, desc: 'meters',      dimension: :length, si: true,  factor: 1)
Unit.new(:km, desc: 'kilometers',  dimension: :length, si: true,  factor: 1000)
Unit.new(:in, desc: 'inches',      dimension: :length, si: false, factor: 0.0254)
Unit.new(:ft, desc: 'feet',        dimension: :length, si: false, factor: 0.0254*12)
Unit.new(:yd, desc: 'yards',       dimension: :length, si: false, factor: 0.0254*36)
Unit.new(:mi, desc: 'miles',       dimension: :length, si: false, factor: 0.0254*12*5280)

# base unit for volume is liters
Unit.new( :ml, desc: 'milliliters',   dimension: :volume, si: true,  factor: 1/1000)
Unit.new( :cl, desc: 'centiliters',   dimension: :volume, si: true,  factor: 1/100)
Unit.new( :dl, desc: 'deciliters',    dimension: :volume, si: true,  factor: 1/10)
Unit.new(  :l, desc: 'liters',        dimension: :volume, si: true,  factor: 1)
Unit.new(:foz, desc: 'fl. ounces',    dimension: :volume, si: false, factor: 3.78541/128)
Unit.new( :cp, desc: 'cups',          dimension: :volume, si: false, factor: 3.78541/16)
Unit.new( :pt, desc: 'pints',         dimension: :volume, si: false, factor: 3.78541/8)
Unit.new( :qt, desc: 'quarts',        dimension: :volume, si: false, factor: 3.78541/4)
Unit.new(:gal, desc: 'us gallons',    dimension: :volume, si: false, factor: 3.78541)

# base unit for mass is grams
Unit.new(  :g, desc: 'grams',       dimension: :mass, si: true,  factor: 1)
Unit.new( :kg, desc: 'kilograms',   dimension: :mass, si: true,  factor: 1000)
Unit.new( :oz, desc: 'ounces',      dimension: :mass, si: false, factor: 28.3495)
Unit.new( :lb, desc: 'pounds',      dimension: :mass, si: false, factor: 28.3495*16)

# base unit for tempurature is celsius
Unit.new( :c, desc: 'celsius',     dimension: :temperature, si: true,  factor: 1)
Unit.new( :f, desc: 'fahrenheit',  dimension: :temperature, si: false, factor: ->(f) { (f - 32) * 5 / 9 },
                                                                      ifactor: ->(c) { c * 9 / 5 + 32 })
# base unit for currency is US dollars
Unit.new(:eur, desc: 'euros',      dimension: :currency, factor: ->(d) { d.convert_currency(:usd, :eur) },
                                                        ifactor: ->(e) { e.convert_currency(:eur, :usd) })
Unit.new(  :€, desc: 'euros',      dimension: :currency, factor: Unit[:eur].factor, ifactor: Unit[:eur].ifactor)
Unit.new(:gbp, desc: 'gb pounds',  dimension: :currency, factor: ->(d) { d.convert_currency(:usd, :gbp) },
                                                        ifactor: ->(p) { p.convert_currency(:gbp, :usd) })
Unit.new(  :£, desc: 'gp pounds',  dimension: :currency, factor: Unit[:gbp].factor, ifactor: Unit[:gbp].ifactor)
Unit.new(:yen, desc: 'yen',        dimension: :currency, factor: ->(d) { d.convert_currency(:usd, :jpy) },
                                                        ifactor: ->(p) { p.convert_currency(:jpy, :usd) })
Unit.new(  :¥, desc: 'yen',        dimension: :currency, factor: Unit[:yen].factor, ifactor: Unit[:yen].ifactor)
Unit.new(:btc, desc: 'bitcoin',    dimension: :currency, factor: ->(d) { d.convert_currency(:usd, :btc) },
                                                        ifactor: ->(p) { p.convert_currency(:btc, :usd) })
Unit.new(:usd, desc: 'us dollars', dimension: :currency, factor: 1)
Unit.new(:'$', desc: 'us dollars', dimension: :currency, factor: 1)

Unit.new(:n, desc: 'numeric (dimensionless)', dimension: nil)

# Denominated ("denominate numbers") are Numeric with optional numerator and/or denominator units
class Denominated
  extend Forwardable

  attr_reader :value, :numerator, :denominator

  def initialize(value, numerator = nil, denominator = nil)
    value = BigDecimal(value, Float::DIG) if value.is_a? Float
    @value = value
    numerator = Unit[numerator] if numerator.is_a? Symbol
    denominator = Unit[denominator ] if denominator.is_a? Symbol
    @numerator = numerator
    @denominator = denominator
  end

  def simplify
    @value = @value.simplify
    self
  end

  def coerce(other)
    [ other, self.value ]
  end

  def units(show_none = false)
    if @numerator.nil? && @denominator.nil?
      show_none ? 'dimensionless' : nil
    else
      "#{@numerator.to_s if @numerator}#{('/' + @denominator.to_s) if @denominator}"
    end
  end

  def to_s(format = nil)
    # default is base 10 with units if present
    return [ to_s(10), to_s(:units) ].compact.join(' ') if format.nil?

    if format == :units
      units
    elsif format == 10 && numerator&.dimension == :time
      case numerator
      when Unit[:hr]
        hours, seconds = (value*3600).divmod(3600)
        minutes, seconds = seconds.divmod(60)
        seconds, frac = seconds.divmod(1)
        output = '%s:%02d:%02d' % [ hours.to_i.grouped($options[:grouped]), minutes, seconds ]
        if frac != 0
          frac = '%.*f' % [ $options[:precision], frac ]        # nicely formatted, but with leading '0.'
          output << frac.sub(/^\d+/, '')
        end
      when Unit[:mn]
        minutes, seconds = (value*60).divmod(60)
        seconds, frac = seconds.divmod(1)
        output = '%s:%02d' % [ minutes.to_i.grouped($options[:grouped]), seconds ]
        if frac != 0
          frac = '%.*f' % [ $options[:precision], frac ]        # nicely formatted, but with leading '0.'
          output << frac.sub(/^\d+/, '')
        end
      else
        output = value.simplify.format(10)
      end
      output
    else
      value.simplify.format(format)
    end
  end

  def apply(unit, denominator_unit = nil)
    raise ArgumentError unless unit.is_a? Unit
    raise ArgumentError if denominator_unit == Unit['n']

    if unit == Unit['n']
      raise ArgumentError if denominator_unit
      @numerator = @denominator = nil
    elsif denominator_unit
      apply(unit)
      apply(denominator_unit)
    elsif @numerator && unit.dimension == @numerator.dimension
      @value = @value.from(numerator, unit)
      @numerator = unit
    elsif @denominator && unit.dimension == @denominator.dimension
      @value = @value.from(unit, denominator)
      @denominator = unit
    elsif @numerator.nil?
      @numerator = unit
    elsif @denominator.nil?
      @denominator = unit
    else
      raise ArgumentError
    end

    self
  end

  def additive(other, op)
    other = Denominated(other) if other.is_a? Numeric

    raise UnitsError, "#{units(true)} #{op} #{other.units(true)}" unless
      numerator&.dimension == other.numerator&.dimension &&
        denominator&.dimension == other.denominator&.dimension

    new_value = if numerator == other.numerator && denominator == other.denominator
                  value.send(op, other.value)
                else
                  value.from(numerator, denominator).send(op, other.value.from(other.numerator, other.denominator))
                       .to(numerator, denominator)
                end

    self.class.new(new_value, numerator, denominator)
  end

  def multiplicative(other, op)
    other = Denominated(other) if other.is_a? Numeric

    # for division
    other.invert if op == :/

    # only supports single dimension numerator and denominator
    if numerator&.dimension == other.denominator&.dimension && denominator&.dimension == other.numerator&.dimension
      new_numerator = new_denominator = nil
    elsif numerator&.dimension == other.denominator&.dimension
      new_numerator = other.numerator
      new_denominator = denominator
    elsif denominator&.dimension == other.numerator&.dimension
      new_numerator = numerator
      new_denominator = other.denominator
    elsif !numerator && !denominator
      new_numerator = other.numerator
      new_denominator = other.denominator
    elsif !other.numerator && !other.denominator
      new_numerator = numerator
      new_denominator = denominator
    else
      other.invert if op == :/
      raise UnitsError, "#{units(true)} #{op} #{other.units(true)}"
    end

    other.invert if op == :/
    new_value = value.from(numerator, denominator).send(op, other.value.from(other.numerator, other.denominator))
                     .to(new_numerator, new_denominator)

    self.class.new(new_value, new_numerator, new_denominator)
  end

  def -(other)
    additive(other, :-)
  end

  def +(other)
    additive(other, :+)
  end

  def <(other)
    additive(other, :<).value
  end

  def >(other)
    additive(other, :>).value
  end

  def *(other)
    multiplicative(other, :*)
  end

  def /(other)
    multiplicative(other, :/)
  end

  def •(other)
    other * self
  end

  define_method(:'.') do |other|
    other * self
  end

  def ÷(other)
    self / other
  end

  def pow(other)
    multiplicative(other, :**)
  end

  def invert
    @numerator, @denominator = @denominator, @numerator
    self
  end

  def reciprocal
    @value = 1/@value
    invert
  end

  # TODO: need to take units into account...
  def <=>(other)
    self < other ? -1 : self > other ? 1 : 0
  end

  def method_missing(symbol, *args)
    raise NoMethodError unless value.respond_to?(symbol)
    raise UnitsError, "#{symbol} is only defined for dimensionless arguments" if
      numerator || denominator || args.any? { |arg| arg.is_a?(Denominated) && (arg.numerator || arg.denominator) }

    value.send(symbol, *args)
  end
end

# convenience method to support Denominated(3.4)
def Denominated(value, numerator = nil, denominator = nil)
  Denominated.new(value, numerator, denominator)
end

class Constant
  attr_reader :name, :desc, :value

  @@instances = { }

  def initialize(name_or_names, desc:, value:, numerator: nil, denominator: nil)
    if name_or_names.is_a? Array
      name_or_names.each { |name| self.class.new(name, desc: desc, value: value, numerator: numerator, denominator: denominator) }
    else
      @name = name_or_names.to_sym
      @desc = desc
      @value = Denominated.new(value, numerator, denominator)
      freeze
      @@instances[name] = self
    end
  end

  def self.all
    @@instances.values
  end

  def self.names
    @@instances.keys
  end

  def self.[](key)
    @@instances[key.to_sym]
  end
end

Constant.new( :C,       desc: 'speed of light', value: 299_792_458, numerator: :m, denominator: :s)
Constant.new([:π, :pi], desc: 'pi', value: Math::PI)
Constant.new(:e,        desc: 'e',  value: Math::E)
Constant.new([:∞, :inf, :infinity], desc: 'infinity', value: Float::INFINITY)

class Stack
  include Math
  extend Forwardable
  def_delegators :@stack, :size, :min, :max, :clear, :empty?

  attr_accessor :formats

  MAGNITUDE = "KMGTPEZY"
  INT = /(?:(?:-?0[xX]\h[,_\h]*) |          # hexadecimal
            (?:-?0[bB][01][,_01]*) |        # binary
            (?:-?\d[,_\d]*))                # decimal
         [#{MAGNITUDE}]?/xo
  IPV4 = /(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})(?!\d)/
  FLOAT = /-?\d[,_\d]*\.\d+([eE]-?\d+)? |   # with decimal point
           -?\d[,_\d]*[eE]-?\d+/x           # with exponent
  ASCII = /'(.*)'/
  TICKER = /@([a-z]+)/i

  TIME_DECIMAL = /-?\d+/
  TIME_UNSIGNED = /\d+/
  TIME_FLOAT = /\d+(?:\.\d+)?/
  HOURS = /(#{TIME_DECIMAL}):(#{TIME_UNSIGNED}):(#{TIME_FLOAT})/o
  MINUTES = /(#{TIME_DECIMAL}):(#{TIME_FLOAT})/o

  REDUCIBLE = /\*\*|[-+\*\.•÷\/&|^]/

  # each unit name is a
  UNITS = Regexp.new(Unit.names.map{|n| n.to_s.sub('$', '\\$') + '(?![A-Za-z])'}.join('|'))
  CONSTANTS = Regexp.new(Constant.names.map{|n| n.to_s + '(?![A-Za-z])'}.join('|'))

  SIGN = { '' => 1, '-' => -1, }
  INPUTS = [
    [ HOURS,                    ->(s) { push Denominated(s[1].int+s[2].int/60.0+s[3].float/3600.0, Unit[:hr]) } ],
    [ MINUTES,                  ->(s) { push Denominated(s[1].int+s[2].float/60.0, Unit[:mn]) } ],
    [ ASCII,                    ->(s) { push s[1].chars.map(&:ord).inject { |acc, op| acc << 8 | op } } ],
    [ TICKER,                   ->(s) { (price = s[1].quote) && push(price) } ],
    [ /(#{INT})\/(#{INT})/o,    ->(s) { push Rational(s[1].int, s[2].int) } ],
    [ IPV4,                     ->(s) { push s[0].ipv4 } ],
    [ FLOAT,                    ->(s) { push s[0].float } ],
    [ INT,                      ->(s) { push s[0].int } ],
    [ /(mean|max|min|size)(!?)/, ->(s) { push stackop(s[1], s[2]) } ],
    [ /@(#{REDUCIBLE})/o,       ->(s) { push reduce(s[1]) } ],
    [ /#{REDUCIBLE}|<<|>>/o,  ->(s) { t = pop; push pop.send(s[0], t) } ],
    [ /(#{UNITS})\/(#{UNITS})/o, ->(s) { push pop.apply(Unit[s[1]], Unit[s[2]]) } ],
    [ /#{UNITS}/o,              ->(s) { push pop.apply(Unit[s[0]]) } ],
    [ /#{CONSTANTS}/o,          ->(s) { push Constant[s[0]].value } ],
    [ /~/,                      ->(s) { push pop.send(s[0]) } ],
    [ /x(?![[:alpha:]])/,       ->(s) { exchange } ],
    [ /round(?![[:alpha:]])/,   ->(s) { push pop.round } ],
    [ /t(runcate)?(?![[:alpha:]])/, ->(s) { push pop.truncate } ],
    [ /!/,                      ->(s) { push pop.factorial } ],
    [ /\[/,                     ->(s) { push pop.floor } ],
    [ /]/,                      ->(s) { push pop.ceil } ],
    [ /rand/,                   ->(s) { push rand } ],
    [ /chs/,                    ->(s) { push (-pop) } ],
    [ /sin|cos|tan|log2|log10|log/, ->(s) { push send(s[0], pop) } ],
    [ /√|sqrt/,                 ->(s) { push pop.sqrt } ],
    [ /p(op)?/,                 ->(s) { pop } ],
    [ /d(up)?/,                 ->(s) { dup } ],
    [ /clear/,                  ->(s) { clear } ],
    [ /r/,                      ->(s) { push pop.reciprocal } ],
    [ /i/,                      ->(s) { push pop.invert } ],
    [ />:([[:alpha:]][[:alnum:]]*)/, ->(s) { reset s[1] } ],
    [ />([[:alpha:]][[:alnum:]]*)/,  ->(s) { set s[1], pop } ],
    [ /<([[:alpha:]][[:alnum:]]*)/,  ->(s) { push get(s[1])} ],
  ]

  def initialize
    @stack = [ ]
    @register = { }
    @formats = [ 10, :units ]
    @last = nil
  end

  def push(arg)
    arg = Denominated.new(arg) unless arg.is_a? Denominated
    @stack.push(arg)
  end

  def pop
    raise ArgumentError if @stack.size == 0
    @last = @stack.pop
  end

  def set(name, arg)
    @register[name] = arg
  end

  def reset(name)
    @register.delete name
  end

  def num_registers
    @register.size
  end

  def get(name)
    raise NameError unless @register.has_key?(name)
    @register[name]
  end

  def dup
    push(@stack[-1].dup)
  end

  def exchange
    @stack[-2], @stack[-1] = @stack[-1], @stack[-2]
  end

  def reduce(op)
    @stack.reduce(op).tap { @stack.clear }
  end

  def mean
    @stack.inject(:+) / @stack.size
  end

  # returns result of doing 'op' on stack, clears stack if modifer is '!'
  def stackop(op, modifier = nil)
    send(op).tap { @stack.clear if modifier == '!' }
  end

  def process(input)
    s = StringScanner.new(input)
    until s.eos?
      s.scan(/\s*/)
      pattern = INPUTS.find { |p| s.scan(p.first) }
      if pattern
        begin
          warn "[#{self}] #{s.matched}" if $options[:trace]
          self.instance_exec(s, &pattern.last)
        rescue RangeError, DomainError => e
          die "Domain error in operation: #{@last} #{s[0]}"
        rescue IPv4Error => e
          die "Invalid IPv4 address #{s[0]}"
        rescue IndexError, ArgumentError => e
          warn caller.inspect if $options[:trace]
          die "Not enough arguments for #{s[0]}: #{e}"
        rescue UnitsError => e
          die "Incompatible units: #{e}"
        rescue NoMethodError => e
          die "#{s[0]} is not defined for this operand: #{@last}: #{e}"
        rescue NameError => e       # N.B. NoMethodError < NameError
          if [ '>', '<' ].include? s.string[0]
            die "Non-existent register '#{s[1]}'"
          else
            die e                   # code bug
          end
        end
      elsif s.pos == 0
        die "Unknown operation: #{s.string}"
      else
        die "Unable to parse here: '#{input.dup.insert(s.pos, red('-->'))}'"
      end
    end
  end

  def to_s
    @stack.map{ |v| v.simplify.to_s }.join(' ')
  end

  def display
    if $options[:oneline]
      puts self unless empty?
    else
      table = [ ]
      @stack.reverse.each do |value|
        table << @formats.map { |fmt| value.to_s(fmt) }
      end

      # right pad the first column to align decimals
      max_frac = table.map { |line| line[0].from_end('.') }.max
      table.each do |line|
        p = line[0].from_end('.')
        line[0] << ' ' * (max_frac - p)
      end

      # one column per format, each starts with width 0
      # first 2 columns are always 10 and :units, last is factor (if requested)
      widths = Array.new(@formats.length+1, 0)
      widths = table.inject(widths) do |current, line|
        line.map { |v| v.to_s.length }.   # map each value to its width
             zip(current).                # zip with current widths
             map { |w| w.max }            # return max of previous, current width
      end
      widths[1] *= -1                     # for units field
      widths[-1] *= -1 if @formats.last == :factor

      table.each do |line|
        puts ("%*s "*line.size) % widths.zip(line).flatten
      end
    end
  end

  @@quotes = nil
  def self.quotes(cur, open)
    unless @@quotes
      @@quotes = [
        [ '', "#{cur} last", 'Δ', 'Δ%', 'low', 'high', '52Wlo', '52Whi', 'ytdΔ%', 'cap', 'pe', open ? green('open') : '' ]
      ]
      @@quotes.first << '' if $options[:verbose] > 1
    end

    @@quotes
  end

  def show_quotes
    print_table(@@quotes, $stderr) if @@quotes
  end

  def show_registers
    puts unless $options[:quiet] || @register.size == 0

    width = @register.keys.map{ |k| k.length }.max
    @register.each do |name, value|
      puts "%*s: #{value.simplify}" % [width, name]
    end
  end

  def stats
    out = {
      'min'    => @stack.min,
      'mean'   => @stack.mean,
      '95%'    => @stack.percentile(95),
      'max'    => @stack.max,
      'stddev' => @stack.standard_deviation,
      'count'  => @stack.length
    }

    prec = 2
    out.each do |key, value|
      out[key] = if value.finite?
                   case value
                   when Integer then '%d%*s' % [ value, prec+1, '' ]
                   when Float   then '%.*f' % [ prec, value ]
                   else
                     value.to_s
                   end
                 else
                   '%s%*s' % [ value, prec+1, '' ]
                 end
    end

    w1 = out.keys.map(&:length).max
    w2 = out.values.map(&:length).max

    out.each do |key, value|
      puts "%*s: %*s" % [ w1, key, w2, value ]
    end
  end
end

def parse(argv, options)
  args = [ ]
  opts = { }
  options_done = false

  OPTIONS.each { |opt| opts[opt[0]] = { type: opt[1], action: opt[3] } }

  while arg = argv.shift
    if arg == '--'
      options_done = true
    elsif arg =~ /-.+/ && Stack::FLOAT !~ arg && Stack::INT !~ arg && !options_done
      raise ArgumentError.new("Unknown option #{arg}") if !opts[arg]
      type = opts[arg][:type]
      raise ArgumentError.new("Missing argument for option #{arg}") if type && argv.empty?
      value = argv.shift if type

      opts[arg][:action].call(options) if type.nil?
      opts[arg][:action].call(options, Integer(value)) if type == Integer
      opts[arg][:action].call(options, Date.parse(value)) if type == Date
      opts[arg][:action].call(options, Regexp.new(value.sub(%r{^/},'').sub(%r{/$},''))) if type == Regexp
    else
      args << arg
    end
  end

  args
rescue ArgumentError => e
  usage
  die e
end

if __FILE__ == $0
  args = parse(ARGV, $options)

  if args.empty? && STDIN.tty?
    usage
    exit
  end

  stack = Stack.new
  stack.formats << 2 if $options[:binary]
  stack.formats << 16 if $options[:hex]
  stack.formats << :ipv4 if $options[:ipv4]
  stack.formats << :ascii if $options[:ascii]
  stack.formats << :factor if $options[:factor]     # assumed to be last

  begin
    # process any input from stdin first
    unless STDIN.tty?
      column = $options[:column]
      column -= 1 if column && column > 0
      delimiter = $options[:delimiter] || ' '

      while line = STDIN.gets
        line.chomp!
        line = line.split(delimiter)[column] if column
        stack.process(line) if line
      end
    end

    # then process each command line argument
    args.each do |arg|
      stack.process(arg)
    end

    stack.show_quotes
    stack.display unless $options[:quiet]
    stack.show_registers

    if $options[:stats]
      puts unless $options[:quiet] && stack.num_registers == 0
      stack.stats
    end
  rescue => e
    puts e.class, e
    raise
  end
end
