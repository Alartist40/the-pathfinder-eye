use embedded_graphics::{
    mono_font::{ascii::FONT_6X10, MonoTextStyleBuilder},
    pixelcolor::BinaryColor,
    prelude::*,
    text::Text,
};
use linux_embedded_hal::I2cdev;
use ssd1306::{prelude::*, I2CDisplayInterface, Ssd1306};
use sysinfo::{System, SystemExt, CpuExt};
use std::{process::Command, thread, time::Duration};

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let i2c = I2cdev::new("/dev/i2c-1")?;
    let interface = I2CDisplayInterface::new(i2c);
    let mut display = Ssd1306::new(interface, DisplaySize128x32, DisplayRotation::Rotate0)
        .into_buffered_graphics_mode();
    display.init().unwrap();

    let text_style = MonoTextStyleBuilder::new()
        .font(&FONT_6X10)
        .text_color(BinaryColor::On)
        .build();

    let mut sys = System::new_all();

    loop {
        sys.refresh_all();
        
        let ip = Command::new("sh")
            .arg("-c")
            .arg("hostname -I | cut -d' ' -f1")
            .output()
            .map(|o| String::from_utf8_lossy(&o.stdout).trim().to_string())
            .unwrap_or_else(|_| "?.?.?.?".to_string());

        let cpu = sys.global_cpu_info().cpu_usage() as u32;
        let mem = (sys.used_memory() as f64 / sys.total_memory() as f64 * 100.0) as u32;
        
        let temp = Command::new("vcgencmd")
            .arg("measure_temp")
            .output()
            .map(|o| String::from_utf8_lossy(&o.stdout).replace("temp=", "").trim().to_string())
            .unwrap_or_else(|_| "0.0'C".to_string());

        display.clear(BinaryColor::Off).unwrap();
        
        Text::new(&format!("IP: {}", ip), Point::new(0, 8), text_style).draw(&mut display).unwrap();
        Text::new(&format!("CPU: {}%", cpu), Point::new(0, 18), text_style).draw(&mut display).unwrap();
        Text::new(&format!("Temp: {}", temp), Point::new(64, 18), text_style).draw(&mut display).unwrap();
        Text::new(&format!("RAM: {}%", mem), Point::new(0, 28), text_style).draw(&mut display).unwrap();
        
        display.flush().unwrap();
        thread::sleep(Duration::from_secs(3));
    }
}
