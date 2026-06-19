mod detection;
mod face_recognition;

use opencv::prelude::*;
use opencv::videoio;
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

fn main() -> Result<(), Box<dyn std::error::Error>> {
    eprintln!("Starting THE-PATHFINDER-EYE Vision Engine v7.2 (Cloud-Ready Mode)");

    std::fs::create_dir_all(CAPTURES_DIR).ok();

    let mut yolo = YOLODetector::new("/home/pi/the-pathfinder-eye_ai/models/yolov5s-640.onnx")?;
    let mut face_detector = FaceDetector::new("/home/pi/the-pathfinder-eye_ai/models/haarcascade_frontalface_default.xml")?;

    let face_rec_model = "/home/pi/the-pathfinder-eye_ai/models/face_recognition.onnx";
    let mut face_recognizer = if Path::new(face_rec_model).exists() {
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

        let mut all_detections = yolo.detect(&frame).unwrap_or_default();
        let face_dets = face_detector.detect(&frame).unwrap_or_default();

        if let Some(ref mut recognizer) = face_recognizer {
            for face_det in &face_dets {
                let roi = core::Rect::new(
                    face_det.bbox[0], face_det.bbox[1],
                    face_det.bbox[2], face_det.bbox[3],
                );
                if let Ok(face_roi) = Mat::roi(&frame, roi) {
                    if let Ok(embedding) = recognizer.extract_embedding(&face_roi) {
                        let known_faces_path = "/home/pi/the-pathfinder-eye_ai/config/known_faces.json";
                        if let Ok(known) = load_known_faces(known_faces_path) {
                            let mut best_name = String::from("face");
                            let mut best_sim = 0.6f32;
                            for (name, known_emb) in &known {
                                let sim = FaceRecognizer::compare_embeddings(&embedding, known_emb);
                                if sim > best_sim {
                                    best_sim = sim;
                                    best_name = format!("face:{}", name);
                                }
                            }
                            all_detections.push(detection::Detection {
                                class_id: -1,
                                class_name: best_name,
                                confidence: best_sim,
                                bbox: face_det.bbox,
                            });
                        }
                    }
                }
            }
        } else {
            all_detections.extend(face_dets);
        }

        let detection_frame = DetectionFrame {
            timestamp: Local::now().to_rfc3339(),
            frame_number: frame_count,
            detections: all_detections.clone(),
            fps: 2.0,
            inference_time_ms: loop_start.elapsed().as_millis() as u64,
        };

        let json_data = serde_json::to_string(&detection_frame)?;
        write_atomic(OUTPUT_JSON_PATH, &json_data)?;

        if last_disk_save.elapsed() > Duration::from_millis(500) {
            imgcodecs::imwrite(OUTPUT_IMAGE_PATH, &frame, &core::Vector::default())?;
            last_disk_save = Instant::now();
        }

        let has_bird = all_detections.iter().any(|d| d.class_name.to_lowercase().contains("bird"));
        let has_face = all_detections.iter().any(|d| d.class_name.starts_with("face"));

        if (has_bird || has_face) && last_capture_time.elapsed() > Duration::from_secs(30) {
            let label = if has_bird { "bird" } else { "face" };
            let filename = format!("{}/{}_{}.jpg", CAPTURES_DIR, label, Local::now().format("%Y%m%d_%H%M%S"));
            imgcodecs::imwrite(&filename, &frame, &core::Vector::default())?;
            last_capture_time = Instant::now();
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

fn load_known_faces(path: &str) -> Result<Vec<(String, ndarray::Array1<f32>)>, Box<dyn std::error::Error>> {
    let data = std::fs::read_to_string(path)?;
    let parsed: std::collections::HashMap<String, Vec<f32>> = serde_json::from_str(&data)?;
    let mut result = Vec::new();
    for (name, emb) in parsed {
        result.push((name, ndarray::Array1::from(emb)));
    }
    Ok(result)
}
