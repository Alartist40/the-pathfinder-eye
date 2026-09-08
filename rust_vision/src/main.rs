mod detection;
mod face_recognition;

use opencv::prelude::*;
use opencv::core::{self, Mat};
use opencv::imgcodecs;
use std::time::{Instant, Duration};
use std::path::Path;
use chrono::Local;
use detection::{YOLODetector, FaceDetector, DetectionFrame};
use face_recognition::FaceRecognizer;
use std::thread;

const OUTPUT_IMAGE_PATH: &str = "/tmp/vision_feed.jpg";
const OUTPUT_JSON_PATH: &str = "/tmp/detections.json";
const CAPTURES_DIR: &str = "/home/pi/the-pathfinder-eye_ai/captures";
const CAPTURE_BIN: &str = "v4l2-ctl";

fn capture_frame_jpeg() -> Result<Vec<u8>, Box<dyn std::error::Error>> {
    let output = std::process::Command::new(CAPTURE_BIN)
        .args(["--device", "/dev/video0", "--set-fmt-video=width=640,height=480,pixelformat=MJPG", "--stream-mmap", "--stream-count=1", "--stream-to=-"])
        .output()?;
    if output.status.success() && !output.stdout.is_empty() {
        Ok(output.stdout)
    } else {
        Err(format!("v4l2-ctl failed: {}", String::from_utf8_lossy(&output.stderr)).into())
    }
}

fn main() -> Result<(), Box<dyn std::error::Error>> {
    eprintln!("Starting THE-PATHFINDER-EYE Vision Engine v7.2 (Cloud-Ready Mode)");

    std::fs::create_dir_all(CAPTURES_DIR).ok();

    let mut yolo = YOLODetector::new("/home/pi/the-pathfinder-eye_ai/models/yolov5s-640.onnx")?;
    let mut face_detector = FaceDetector::new("/home/pi/the-pathfinder-eye_ai/models/haarcascade_frontalface_default.xml")?;

    let face_rec_model = "/home/pi/the-pathfinder-eye_ai/models/face_recognition.onnx";
    let _face_recognizer = if Path::new(face_rec_model).exists() {
        eprintln!("Face recognition model found, initializing...");
        match FaceRecognizer::new(face_rec_model) {
            Ok(r) => {
                eprintln!("Face recognizer initialized");
                Some(r)
            }
            Err(e) => {
                eprintln!("Failed to init face recognizer (non-fatal): {}", e);
                None
            }
        }
    } else {
        eprintln!("No face recognition model at {}, skipping recognition", face_rec_model);
        None
    };

    let mut frame_count: u64 = 0;
    let mut last_capture_time = Instant::now();
    let mut last_disk_save = Instant::now();

    loop {
        let loop_start = Instant::now();

        let jpeg_data = match capture_frame_jpeg() {
            Ok(d) => d,
            Err(e) => {
                eprintln!("Capture error: {}, retrying...", e);
                thread::sleep(Duration::from_millis(500));
                continue;
            }
        };

        let frame = match imgcodecs::imdecode(&core::Vector::from_slice(&jpeg_data), imgcodecs::IMREAD_COLOR) {
            Ok(f) => f,
            Err(e) => {
                eprintln!("Decode error: {}", e);
                thread::sleep(Duration::from_millis(100));
                continue;
            }
        };

        eprintln!("Frame captured, running detection...");
        let detect_start = Instant::now();
        let mut all_detections = yolo.detect(&frame).unwrap_or_default();
        eprintln!("YOLO done in {:?}", detect_start.elapsed());
        let face_dets = face_detector.detect(&frame).unwrap_or_default();
        eprintln!("Face detect done, total={}", all_detections.len() + face_dets.len());

        for face_det in &face_dets {
            all_detections.push(detection::Detection {
                class_id: -1,
                class_name: "face".to_string(),
                confidence: 0.6,
                bbox: face_det.bbox,
            });
        }

        eprintln!("Writing output...");
        let json_result = serde_json::to_string(&DetectionFrame {
            timestamp: Local::now().to_rfc3339(),
            frame_number: frame_count,
            detections: all_detections.clone(),
            fps: 2.0,
            inference_time_ms: loop_start.elapsed().as_millis() as u64,
        });
        if let Ok(json_data) = json_result {
            eprintln!("STEP: writing detections...");
            write_atomic(OUTPUT_JSON_PATH, &json_data).ok();
            if last_disk_save.elapsed() > Duration::from_millis(500) {
                imgcodecs::imwrite(OUTPUT_IMAGE_PATH, &frame, &core::Vector::default()).ok();
                last_disk_save = Instant::now();
            }

            let has_bird = all_detections.iter().any(|d| d.class_name.to_lowercase().contains("bird"));
            let has_face = all_detections.iter().any(|d| d.class_name.starts_with("face"));

            if (has_bird || has_face) && last_capture_time.elapsed() > Duration::from_secs(30) {
                let label = if has_bird { "bird" } else { "face" };
                let filename = format!("{}/{}_{}.jpg", CAPTURES_DIR, label, Local::now().format("%Y%m%d_%H%M%S"));
                imgcodecs::imwrite(&filename, &frame, &core::Vector::default()).ok();
                last_capture_time = Instant::now();
            }
        }

        frame_count += 1;

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

