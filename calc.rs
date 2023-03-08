use std::env;

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
                // Pop the top two values off the stack
                let b = stack.pop().expect("Stack underflow");
                let a = stack.pop().expect("Stack underflow");

                // Perform the operation and push the result back onto the stack
                let result = match arg.as_str() {
                    "+" => a + b,
                    "-" => a - b,
                    "*" => a * b,
                    "/" => a / b,
                    "%" => a % b,
                    _ => panic!("Invalid operator"),
                };
                stack.push(result);
            }
        }
    }

    // Print the top value on the stack
    if let Some(result) = stack.pop() {
        println!("{}", result);
    } else {
        println!("Stack is empty");
    }
}
