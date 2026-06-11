mod detection;
mod face_recognition;

use opencv::{
    prelude::*,
    videoio,
    core,
    imgcodecs,
};
use std::time::{Instant, Duration};
use std::path::Path;
use chrono::Local;
use detection::{YOLODetector, FaceDetector, DetectionFrame};
use std::thread;

const OUTPUT_IMAGE_PATH: &str = "/tmp/vision_feed.jpg";
const OUTPUT_JSON_PATH: &str = "/tmp/detections.json";
const CAPTURES_DIR: &str = "/home/pi/the-pathfinder-eye_ai/captures";

fn main() -> Result<(), Box<dyn std::error::Error>> {
    eprintln!("🚀 Starting THE-PATHFINDER-EYE Vision Engine v7.1 (Cloud-Ready Mode)...!");
    
    std::fs::create_dir_all(CAPTURES_DIR).ok();

    // 1. Initialize Detectors
    let mut yolo = YOLODetector::new("/home/pi/the-pathfinder-eye_ai/models/yolov5s-640.onnx")?;
    let mut face_detector = FaceDetector::new("/home/pi/the-pathfinder-eye_ai/models/haarcascade_frontalface_default.xml")?;
    
    // 2. Open Camera
    let mut cam = videoio::VideoCapture::new(0, videoio::CAP_ANY)?;
    cam.set(videoio::CAP_PROP_FRAME_WIDTH, 640.0)?;
    cam.set(videoio::CAP_PROP_FRAME_HEIGHT, 480.0)?;
    
    let mut frame_count: u64 = 0;
    let mut last_capture_time = Instant::now();
    let mut last_disk_save = Instant::now();
    
    loop {
        let loop_start = Instant::now();
        let mut frame = Mat::default();
        
        if !cam.read(&mut frame)? || frame.empty() { continue; }
        
        // --- SENSE & THINK ---
        let mut all_detections = yolo.detect(&frame).unwrap_or_default();
        let face_dets = face_detector.detect(&frame).unwrap_or_default();
        all_detections.extend(face_dets);
        
        // --- ACT (JSON Output) ---
        let detection_frame = DetectionFrame {
            timestamp: Local::now().to_rfc3339(),
            frame_number: frame_count,
            detections: all_detections.clone(),
            fps: 2.0, 
            inference_time_ms: loop_start.elapsed().as_millis() as u64,
        };
        
        let json_data = serde_json::to_string(&detection_frame)?;
        write_atomic(OUTPUT_JSON_PATH, &json_data)?;
        
        // --- SAVE TO DISK FOR AI (2 FPS) ---
        if last_disk_save.elapsed() > Duration::from_millis(500) {
            imgcodecs::imwrite(OUTPUT_IMAGE_PATH, &frame, &core::Vector::default())?;
            last_disk_save = Instant::now();
        }

        // --- EVENT TRIGGERED PERSISTENT CAPTURE ---
        let has_bird = all_detections.iter().any(|d| d.class_name.to_lowercase().contains("bird"));
        let has_face = all_detections.iter().any(|d| d.class_name.to_lowercase().contains("face"));
        
        if (has_bird || has_face) && last_capture_time.elapsed() > Duration::from_secs(30) {
            let label = if has_bird { "bird" } else { "face" };
            let filename = format!("{}/{}_{}.jpg", CAPTURES_DIR, label, Local::now().format("%Y%m%d_%H%M%S"));
            imgcodecs::imwrite(&filename, &frame, &core::Vector::default())?;
            last_capture_time = Instant::now();
        }
        
        frame_count += 1;
        
        // --- CPU THROTTLE (2 FPS) ---
        let target_duration = Duration::from_millis(500);
        let elapsed = loop_start.elapsed();
        if elapsed < target_duration {
            thread::sleep(target_duration - elapsed);
        }
    }
}

fn write_atomic(path: &str, content: &str) -> std::io::Result<()> {
    let temp_path = format!("{}.tmp", path);
    std::fs::write(&temp_path, content)?;
    std::fs::rename(&temp_path, path)
}
