use std::env;
use std::process;

struct ValueWithUnits {
    value: f64,
    numerator: String,
    denominator: String,
}

impl ValueWithUnits {
    fn new(value: f64, numerator: &str, denominator: &str) -> Self {
        Self {
            value,
            numerator: numerator.to_owned(),
            denominator: denominator.to_owned(),
        }
    }

    fn to_string(&self) -> String {
        format!("{} {} / {}", self.value, self.numerator, self.denominator)
    }
}

fn main() {
    let mut stack: Vec<ValueWithUnits> = Vec::new();

    // Loop through each argument provided
    for arg in env::args().skip(1) {
        // Try to parse the argument as a floating point number
        match arg.parse::<f64>() {
            // If it's a number, push it onto the stack
            Ok(num) => {
                stack.push(ValueWithUnits::new(num, "", ""));
            }
            // If it's not a number, assume it's an operator
            Err(_) => {
                match arg.as_str() {
                    "+" => {
                        let b = stack.pop().expect("Stack underflow");
                        let a = stack.pop().expect("Stack underflow");
                        stack.push(ValueWithUnits::new(
                            a.value + b.value,
                            a.numerator.as_str(),
                            a.denominator.as_str(),
                        ));
                    }
                    "-" => {
                        let b = stack.pop().expect("Stack underflow");
                        let a = stack.pop().expect("Stack underflow");
                        stack.push(ValueWithUnits::new(
                            a.value - b.value,
                            a.numerator.as_str(),
                            a.denominator.as_str(),
                        ));
                    }
                    "*" | "." => {
                        let b = stack.pop().expect("Stack underflow");
                        let a = stack.pop().expect("Stack underflow");
                        stack.push(ValueWithUnits::new(
                            a.value * b.value,
                            format!("{} {}", a.numerator, b.numerator).as_str(),
                            format!("{} {}", a.denominator, b.denominator).as_str(),
                        ));
                    }
                    "/" => {
                        let b = stack.pop().expect("Stack underflow");
                        let a = stack.pop().expect("Stack underflow");
                        stack.push(ValueWithUnits::new(
                            a.value / b.value,
                            format!("{} {}", a.numerator, b.denominator).as_str(),
                            format!("{} {}", a.denominator, b.numerator).as_str(),
                        ));
                    }
                    "%" => {
                        let b = stack.pop().expect("Stack underflow");
                        let a = stack.pop().expect("Stack underflow");
                        stack.push(ValueWithUnits::new(
                            a.value % b.value,
                            "",
                            "",
                        ));
                    }
                    "mean" => {
                        let sum: f64 = stack.iter().map(|v| v.value).sum();
                        let count = stack.len() as f64;
                        let avg = sum / count;
                        stack.clear();
                        stack.push(ValueWithUnits::new(avg, "", ""));
                    }
                    _ => {
                        println!("Unknown operator: {}", arg);
                        process::exit(1);
                    }
                };
            }
        }
    }

    // Print each element of the stack on its own line
    for val in stack.iter().rev() {
        println!("{}", val.to_string());
    }
}
