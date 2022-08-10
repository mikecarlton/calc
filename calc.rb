#!/usr/bin/env ruby -w

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

  puts <<EOS

Input is read from stdin and then the command line.

Numbers (with optional leading '-'; can use ',' or '_' to group digits):
    Integers: decimal, binary (leading 0b), or hexadecimal (leading 0x)
    Rationals: <integer>/<integer>
    Floats: decimal, with optional exponent (E)

    Integers can have a final binary magnitude factor (KMGTPEZY) for
    kilo-, mega-, giga-, tera-, peta-, exa-, zetta- or yotta-byte

Arithmetic operations (prepend '@' to reduce the stack):
    ** - + * • m / ÷ %
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
    p π
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

$options = { }
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

    Float(self.gsub(/[,_]/, ''))
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

Numeric.class_eval do
  define_method(:•) do |other|
    other * self
  end

  define_method(:m) do |other|
    other * self
  end

  define_method(:÷) do |other|
    self / other
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
      original_div.bind(self).call(Float(other))
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

  REDUCIBLE = /\*\*|[-+m•*÷\/&|^]|lcm|gcd/

  SIGN = { '' => 1, '-' => -1, }
  INPUTS = [
    [ /(#{INT})\/(#{INT})/o,    ->(s) { push Rational(s[1].int, s[2].int) } ],
    [ IPV4,                     ->(s) { push s[0].ipv4 } ],
    [ FLOAT,                    ->(s) { push s[0].float } ],
    [ INT,                      ->(s) { push s[0].int } ],
    [ /(-?)(π|pi)(?![[:alnum:]])/i, ->(s) { push SIGN[s[1]] * PI } ],
    [ /(-?)e(?![[:alnum:]])/i,      ->(s) { push SIGN[s[1]] * E } ],
    [ /(-?)(∞|inf(inity)?)(?![[:alnum:]])/i, ->(s) { push SIGN[s[1]] * Float::INFINITY } ],
    [ /@(#{REDUCIBLE})/o,       ->(s) { push reduce(s[1]) } ],
    [ /(#{REDUCIBLE})|<<|>>/,   ->(s) { t = pop; push pop.send(s[0], t) } ],
    [ /~/,                      ->(s) { push pop.send(s[0]) } ],
    [ /x(?![[:alpha:]])/,       ->(s) { exchange } ],
    [ /round(?![[:alpha:]])/,   ->(s) { push pop.round } ],
    [ /t(runcate)?(?![[:alpha:]])/, ->(s) { push pop.truncate } ],
    [ /!/,                      ->(s) { push pop.factorial } ],
    [ /\[/,                     ->(s) { push pop.floor } ],
    [ /]/,                      ->(s) { push pop.ceil } ],
    [ /(mean|max|min|size)(!?)/, ->(s) { push stackop(s[1], s[2]) } ],
    [ /rand/,                   ->(s) { push rand } ],
    [ /chs/,                    ->(s) { push (-pop) } ],
    [ /sin|cos|tan|log2|log10|log/, ->(s) { push send(s[0], pop) } ],
    [ /√|sq(rt)?/,              ->(s) { push pop.sqrt } ],
    [ /p(op)?/,                 ->(s) { pop } ],
    [ /d(up)?/,                 ->(s) { dup } ],
    [ /clear/,                  ->(s) { clear } ],
    [ /r/,                      ->(s) { push 1/pop } ],
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
    push(@stack[-1])
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
          die "Not enough arguments for #{s[0]}"
        rescue TypeError, NoMethodError => e
          die "Not defined for this operand: #{@last} #{s[0]}"
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
    @stack.map{ |v| v.simplify }.join(' ')
  end

  def display
    if $options[:oneline]
      puts self unless empty?
    else
      table = [ ]
      @stack.reverse.each do |value|
        table << @formats.map { |fmt| value.simplify.format(fmt) }
      end

      # one column per format, each starts with width 0
      widths = Array.new(@formats.length, 0)
      widths = table.inject(widths) do |current, line|
        line.map { |v| v.length }.   # map each value to its width
             zip(current).           # zip with current widths
             map { |w| w.max }       # return max of previous, current width
      end
      widths[-1] *= -1 if @formats.last == :factor

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

begin
  $parser.order!(ARGV)
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
