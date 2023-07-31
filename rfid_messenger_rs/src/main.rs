extern crate hidapi;

use hidapi::HidApi;

fn main() {
    println!("Printing all available hid devices:");

    match HidApi::new() {
        Ok(api) => {
            for device in api.device_list() {
                println!(
                    "Vendor Id: {:04x} , Product Id: {:04x},Product String {:?}",
                    device.vendor_id(),
                    device.product_id(),
                    device.product_string().unwrap()
                );
            }
        }
        Err(e) => {
            eprintln!("Error: {}", e);
        }
    }
}
