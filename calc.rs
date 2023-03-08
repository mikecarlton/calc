use std::env;
use std::process;

fn main() {
    let mut stack: Vec<f64> = Vec::new();

    // Loop through each argument provided
    for arg in env::args().skip(1) {
        // Try to parse the argument as a floating point number
        match arg.parse::<f64>() {
            // If it's a number, push it onto the stack
            Ok(num) => stack.push(num),
            // If it's not a number, assume it's an operator
            Err(_) => {
                match arg.as_str() {
                    "+" => {
                        let b = stack.pop().expect("Stack underflow");
                        let a = stack.pop().expect("Stack underflow");
                        stack.push(a + b);
                    }
                    "-" => {
                        let b = stack.pop().expect("Stack underflow");
                        let a = stack.pop().expect("Stack underflow");
                        stack.push(a - b);
                    }
                    "*" | "." => {
                        let b = stack.pop().expect("Stack underflow");
                        let a = stack.pop().expect("Stack underflow");
                        stack.push(a * b);
                    }
                    "/" => {
                        let b = stack.pop().expect("Stack underflow");
                        let a = stack.pop().expect("Stack underflow");
                        stack.push(a / b);
                    }
                    "%" => {
                        let b = stack.pop().expect("Stack underflow");
                        let a = stack.pop().expect("Stack underflow");
                        stack.push(a % b);
                    }
                    "mean" => {
                        let sum: f64 = stack.iter().sum();
                        let count = stack.len() as f64;
                        stack.clear();
                        stack.push(sum / count);
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
    for num in stack.iter().rev() {
        println!("{}", num);
    }
}
