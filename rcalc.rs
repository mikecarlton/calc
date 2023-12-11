use std::collections::HashMap;
use std::env;

struct Unit {
    value: f64,
    numerator: String,
    denominator: String,
    factor: f64,
}

impl Unit {
    fn new(value: f64, numerator: String, denominator: String, factor: f64) -> Self {
        Unit {
            value,
            numerator,
            denominator,
            factor,
        }
    }

    fn convert(&self, other: &Unit) -> (f64, String, String) {
        let numerator = if self.numerator == other.numerator {
            String::from(&other.numerator)
        } else {
            format!("{}{}", other.numerator, self.numerator)
        };
        let denominator = if self.denominator == other.denominator {
            String::from(&other.denominator)
        } else {
            format!("{}{}", self.denominator, other.denominator)
        };
        let factor = other.factor * self.factor;
        (self.value * factor / other.value, numerator, denominator)
    }
}

struct Calculator {
    stack: Vec<Unit>,
    conversions: HashMap<String, f64>,
}

impl Calculator {
    fn new(conversions: HashMap<String, f64>) -> Self {
        Calculator {
            stack: Vec::new(),
            conversions,
        }
    }

    fn push(&mut self, value: f64, numerator: String, denominator: String, factor: f64) {
        let unit = Unit::new(value, numerator, denominator, factor);
        self.stack.push(unit);
    }

    fn pop(&mut self) -> Option<Unit> {
        self.stack.pop()
    }

    fn add(&mut self) {
        if let (Some(a), Some(b)) = (self.pop(), self.pop()) {
            let (value, numerator, denominator) = b.convert(&a);
            self.push(value, numerator, denominator, a.factor);
        } else {
            eprintln!("Error: not enough operands");
        }
    }

    fn sub(&mut self) {
        if let (Some(a), Some(b)) = (self.pop(), self.pop()) {
            let (value, numerator, denominator) = b.convert(&a);
            self.push(value, numerator, denominator, a.factor);
        } else {
            eprintln!("Error: not enough operands");
        }
    }

    fn mul(&mut self) {
        if let (Some(a), Some(b)) = (self.pop(), self.pop()) {
            let value = a.value * b.value;
            let numerator = format!("{}{}", a.numerator, b.numerator);
            let denominator = format!("{}{}", a.denominator, b.denominator);
            let factor = a.factor * b.factor;
            self.push(value, numerator, denominator, factor);
        } else {
            eprintln!("Error: not enough operands");
        }
    }

    fn div(&mut self) {
        if let (Some(a), Some(b)) = (self.pop(), self.pop()) {
            let value = b.value / a.value;
            let numerator = format!("{}{}", a.denominator, b.numerator);
            let denominator = format!("{}{}", a.numerator, b.denominator);
            let factor = b.factor / a.factor;
            self.push(value, numerator, denominator, factor);
        } else {
            eprintln!("Error: not enough operands");
        }
    }

    fn mean(&mut self) {
        let sum = self.stack.iter().fold(0.0, |







fn main() {
    let mut conversions = HashMap::new();
    conversions.insert(String::from("km"), 1000.0);
    conversions.insert(String::from("m"), 1.0);
    conversions.insert(String::from("cm"), 0.01);
    conversions.insert(String::from("mm"), 0.001);
    conversions.insert(String::from("mi"), 1609.344);
    conversions.insert(String::from("yd"), 0.9144);
    conversions.insert(String::from("ft"), 0.3048);
    conversions.insert(String::from("in"), 0.0254);

    let args: Vec<String> = env::args().collect();
    let mut calculator = Calculator::new(conversions);

    for arg in args.iter().skip(1) {
        match arg.as_str() {
            "+" => calculator.add(),
            "-" => calculator.sub(),
            "*" | "." => calculator.mul(),
            "/" => calculator.div(),
            "mean" => calculator.mean(),
            s => {
                if let Ok(value) = s.parse::<f64>() {
                    calculator.push(value, String::new(), String::new(), 1.0);
                } else {
                    let len = s.len();
                    let (numerator, denominator) = s.split_at(len - 2);
                    if let Some(&factor) = calculator.conversions.get(denominator) {
                        calculator.push(1.0, String::from(numerator), String::from(denominator), factor);
                    } else {
                        eprintln!("Error: unknown operator {}", s);
                        return;
                    }
                }
            }
        }
    }

    for unit in calculator.stack.iter() {
        println!("{:.4} {}/{}", unit.value, unit.numerator, unit.denominator);
    }
}
