#!/usr/bin/env ruby

=begin
MIT License

Copyright (c) 2017 Mike Carlton

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

require 'optparse'
require 'strscan'
require 'forwardable'
require 'bigdecimal'

OPTIONS =
[
  [ "-t", "--trace", "Trace operations",
    ->(v) { $options[:trace] = v } ],
  [ "-b", "--binary", "Show binary representation of integers",
    ->(v) { $options[:binary] = v } ],
  [ "-x", "--hex", "Show hex representation of integers",
    ->(v) { $options[:hex] = v } ],
  [ "-i", "--ip", "Show IPv4 representation of integers",
    ->(v) { $options[:ipv4] = v } ],
  [ "-c", "--column INTEGER", Integer, "Column to extract from lines on stdin (negative counts from end)",
    ->(v) { raise OptionParser::InvalidOption.new("cannot be 0") if v == 0
            $options[:column] = v } ],
  [ "-p", "--precision INTEGER", Integer, "Set display precision for floating point number)",
    ->(v) { raise OptionParser::InvalidOption.new("cannot be negative") if v < 0
            $options[:precision] = v } ],
  [ "-d", "--delimiter REGEXP", Regexp, "Regular expression to split columns (default: whitespace)",
    ->(v) { $options[:delimiter] = v } ],
  [ "-g", "--group", "Use ',' to group decimal numbers",
    ->(v) { $options[:grouped] = ',' } ],
  [ "-a", "--ascii", "Show ASCII representation of integers",
    ->(v) { $options[:ascii] = v } ],
  [ "-f", "--factor", "Show prime factorization of integers",
    ->(v) { $options[:factor] = v } ],
  [ "-s", "--stats", "Show statistics of values",
    ->(v) { $options[:stats] = v } ],
  [ "-q", "--quiet", "Do not show stack at finish",
    ->(v) { $options[:quiet] = v } ],
  [ "-o", "--oneline", "Show final stack on one line",
    ->(v) { $options[:oneline] = v } ],
  [ "-h", "--help", "Show extended help",
    ->(_) { help ; exit } ],
]

def help
  puts $parser

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

   Units:
      Units are applied if dimensionless, otherwise converted

      #{ Unit.all.map { |unit| "#{unit.name} #{unit.desc}" }.join("\n    ") }
  EOS
end

def red(msg)
  "\033[31m#{msg}\033[0m"
end

def die(*msgs)
  warn(*msgs)
  exit(1)
end

def usage
  die $parser
end

$options = {
  precision: 2
}
$parser = OptionParser.new do |opts|
  opts.banner = "Usage: [ARGUMENTS | ] #{File.basename($0)} [OPTIONS] [--] ARGUMENT(S)"
  opts.separator ""
  opts.separator "Options:"

  OPTIONS.each do |option|
    block = option.pop
    opts.on(*option, &block)
  end
end

class IPv4Error < ArgumentError
end

class UnitsError < TypeError
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

  # parse a string (with possible ',' and/or '_' separators) into float
  def float
    /([_,])/.match(self) { |m| $options[:grouped] ||= m[1] }

    BigDecimal(self.gsub(/[,_]/, ''), Float::DIG)
  end
end

class BigDecimal
  orig_to_s = instance_method :to_s

  # default to printing floating point format instead of exponential
  define_method :to_s do |*param|
    orig_to_s.bind(self).call(param.first || 'F')
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
    value = self
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

  # FIXME: lookup and cache in yaml file from api
  # eu:
  #   usd: 1.1
  #   updated: datetime
  def convert_currency(from, to)
    self * 1.1
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

# Define / to do integer division if exact, else floating
# N.B. Only do this if you know all code expects this behavior
#
klass = RUBY_VERSION.to_f < 2.4 ? Fixnum : Integer # Fixnum deprecated in 2.4
klass.class_eval do
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
  def sum
    reduce(:+)
  end unless defined? Array.new.sum       # already exists in ruby 2.4

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
  attr_reader :name, :desc, :dimension, :factor, :ifactor

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
  def initialize(name, desc:, dimension:, factor:, ifactor: nil)
    @name = name.to_sym
    @desc = desc
    @dimension = dimension
    @factor = factor
    @ifactor = ifactor
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

Unit.new(:mm, desc: 'millimeters', dimension: :length, factor: 1/1000)
Unit.new(:cm, desc: 'centimeters', dimension: :length, factor: 1/100)
Unit.new(:km, desc: 'kilometers',  dimension: :length, factor: 1000)
Unit.new(:in, desc: 'inches',      dimension: :length, factor: 0.0254)
Unit.new(:ft, desc: 'feet',        dimension: :length, factor: 0.0254*12)
Unit.new(:yd, desc: 'yards',       dimension: :length, factor: 0.0254*36)
Unit.new(:mi, desc: 'miles',       dimension: :length, factor: 0.0254*12*5280)
Unit.new( :m, desc: 'meters',      dimension: :length, factor: 1)

Unit.new( :ml, desc: 'milliliters',   dimension: :volume, factor: 1/1000)
Unit.new(  :l, desc: 'liters',        dimension: :volume, factor: 1)
Unit.new(:gal, desc: 'gallons (us)',  dimension: :volume, factor: 3.78541)
Unit.new( :qt, desc: 'quarts',        dimension: :volume, factor: 3.78541/4)
Unit.new(:foz, desc: 'fl. ounces',    dimension: :volume, factor: 3.78541/128)

Unit.new(  :g, desc: 'grams',       dimension: :weight, factor: 1)
Unit.new( :kg, desc: 'kilograms',   dimension: :weight, factor: 1000)
Unit.new( :oz, desc: 'ounces',      dimension: :weight, factor: 28.3495)
Unit.new( :lb, desc: 'pounds',      dimension: :weight, factor: 28.3495*16)

Unit.new( :c, desc: 'celsius',     dimension: :temperature, factor: 1)
Unit.new( :f, desc: 'fahrenheit',  dimension: :temperature, factor: ->(f) { (f - 32) * 5 / 9 },
                                                            ifactor: ->(f) { f * 9 / 5 + 32 })
Unit.new( :eu, desc: 'euros',      dimension: :currency, factor: ->(n) { n.convert_currency(:eu, :usd) })
Unit.new(  :€, desc: 'euros',      dimension: :currency, factor: Unit[:eu].factor)
Unit.new(:usd, desc: 'us dollars', dimension: :currency, factor: 1)
Unit.new(:'$', desc: 'us dollars', dimension: :currency, factor: 1)

Unit.new(:n, desc: 'numeric (dimensionless)', dimension: nil, factor: nil)

# Denominated ("denominate numbers") are Numeric with optional numerator and/or denominator units
class Denominated
  extend Forwardable

  attr_reader :value, :numerator, :denominator
  def_delegators :@value, :simplify

  def initialize(value, numerator = nil, denominator = nil)
    value = BigDecimal(value, Float::DIG) if value.is_a? Float
    @value = value
    @numerator = numerator
    @denominator = denominator
  end

  def coerce(other)
    [ other, self.value ]
  end

  def units(show_none = false)
    @numerator.nil? && @denominator.nil? && show_none ? 'dimensionless' :
      "#{@numerator.to_s if @numerator}#{('/' + @denominator.to_s) if @denominator}" 
  end

  def to_s(format = 10)
    if format == 10 && numerator&.dimension == :time
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
    else
      output = value.simplify.format(format)
    end

    "#{output}#{" #{units}" if units}"
  end

  def formatted(format)
    raise "dead code"
    return to_s(format)

    if format == 10 && numerator&.dimension == :time
      case numerator
      when Unit[:hr]
        hours, seconds = value.divmod(3600)
        minutes, seconds = seconds.divmod(60)
        '%s:%02d:%02f' % [ hours.format(10), minutes, seconds ]
      when Unit[:mn]
        minutes, seconds = value.divmod(60)
        '%s:%02f' % [ minutes.format(10), seconds ]
      else
        value.format(10)
      end
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
    raise ArgumentError unless other.class == self.class
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
    raise ArgumentError unless other.class == self.class

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
    other ** self
  end

  def invert
    @numerator, @denominator = @denominator, @numerator
    self
  end

  def reciprocal
    @value = 1/@value
    invert
  end

  def method_missing(symbol, *args)
    raise NoMethodError unless value.respond_to?(symbol)
    raise UnitsError, "#{symbol} is only defined for dimensionless arguments" if
      numerator || denominator || args.any? { |arg| arg.numerator || arg.denominator }

    value.send(symbol, *args)
  end
end

# convenience method to support Denominated(3.4)
def Denominated(value, numerator = nil, denominator = nil)
  Denominated.new(value, numerator, denominator)
end

class Stack
  include Math
  extend Forwardable
  def_delegators :@stack, :size, :min, :max, :clear, :empty?

  attr_accessor :formats
  attr_accessor :precision

  MAGNITUDE = "KMGTPEZY"
  INT = /(?:(?:-?0[xX]\h[,_\h]*) |          # hexadecimal
            (?:-?0[bB][01][,_01]*) |        # binary
            (?:-?\d[,_\d]*))                # decimal
         [#{MAGNITUDE}]?/xo
  IPV4 = /(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})(?!\d)/
  FLOAT = /-?\d[,_\d]*\.\d+([eE]-?\d+)? |   # with decimal point
           -?\d[,_\d]*[eE]-?\d+/x           # with exponent

  TIME_DECIMAL = /-?\d+/
  TIME_UNSIGNED = /\d+/
  TIME_FLOAT = /\d+(?:\.\d+)?/
  HOURS = /(#{TIME_DECIMAL}):(#{TIME_UNSIGNED}):(#{TIME_FLOAT})/o
  MINUTES = /(#{TIME_DECIMAL}):(#{TIME_FLOAT})/o

  REDUCIBLE = /\*\*|[-+\*\.•÷\/&|^]|lcm|gcd|pow/
  UNITS = Regexp.new(Unit.names.join('|'))

  SIGN = { '' => 1, '-' => -1, }
  INPUTS = [
    [ HOURS,                    ->(s) { push Denominated(s[1].int+s[2].int/60.0+s[3].float/3600.0, Unit[:hr]) } ],
    [ MINUTES,                  ->(s) { push Denominated(s[1].int+s[2].float/60.0, Unit[:mn]) } ],
    [ /(#{INT})\/(#{INT})/o,    ->(s) { push Rational(s[1].int, s[2].int) } ],
    [ IPV4,                     ->(s) { push s[0].ipv4 } ],
    [ FLOAT,                    ->(s) { push BigDecimal(s[0].gsub(/[,_]/, '')) } ],
    [ INT,                      ->(s) { push s[0].int } ],
    [ /(-?)(π|pi)(?![[:alnum:]])/i, ->(s) { push SIGN[s[1]] * PI } ],
    [ /(-?)e(?![[:alnum:]])/i,      ->(s) { push SIGN[s[1]] * E } ],
    [ /(-?)(∞|inf(inity)?)(?![[:alnum:]])/i, ->(s) { push SIGN[s[1]] * Float::INFINITY } ],
    [ /(mean|max|min|size)(!?)/, ->(s) { push stackop(s[1], s[2]) } ],
    [ /@(#{REDUCIBLE})/o,       ->(s) { push reduce(s[1]) } ],
    [ /#{REDUCIBLE}|<<|>>/o,  ->(s) { t = pop; push pop.send(s[0], t) } ],
    [ /(#{UNITS})\/(#{UNITS})/o, ->(s) { push pop.apply(Unit[s[1]], Unit[s[2]]) } ],
    [ /#{UNITS}/o,              ->(s) { push pop.apply(Unit[s[0]]) } ],
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
    [ /√|sq(rt)?/,              ->(s) { push pop.sqrt } ],
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
    @formats = [ 10 ]
    @last = nil
    @precision = nil
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
          puts caller.inspect
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
    @stack.map{ |v| v.simplify.to_s + v.units }.join(' ')
  end

  def display
    if $options[:oneline]
      puts self unless empty?
    else
      table = [ ]
      @stack.reverse.each do |value|
        #table << @formats.map { |fmt| value.simplify.format(fmt) }.append(value.units)
        table << @formats.map { |fmt| value.to_s(fmt) }
      end

      # one column per format plus units, each starts with width 0
      widths = Array.new(@formats.length+1, 0)
      widths = table.inject(widths) do |current, line|
        line.map { |v| v.length }.   # map each value to its width
             zip(current).           # zip with current widths
             map { |w| w.max }       # return max of previous, current width
      end
      widths[-1] *= -1 # for units field
      widths[-2] *= -1 if @formats.last == :factor

      table.each do |line|
        puts ("%*s "*line.size) % widths.zip(line).flatten
      end
    end
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

if __FILE__ == $0
  begin
    $parser.parse!(ARGV)
  rescue OptionParser::InvalidOption => e
    puts e
    usage
    exit
  end

  if ARGV.empty? && STDIN.tty?
    usage
    exit
  end

  stack = Stack.new
  stack.formats << 2 if $options[:binary]
  stack.formats << 16 if $options[:hex]
  stack.formats << :ipv4 if $options[:ipv4]
  stack.formats << :ascii if $options[:ascii]
  stack.formats << :factor if $options[:factor]

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
    ARGV.each do |arg|
      stack.process(arg)
    end

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
