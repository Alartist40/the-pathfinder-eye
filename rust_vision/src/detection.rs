use opencv::{prelude::*, core, imgproc, objdetect};
use serde::{Serialize, Deserialize};
use anyhow::Result;

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Detection {
    pub class_id: i32,
    pub class_name: String,
    pub confidence: f32,
    pub bbox: [i32; 4],  // [x, y, w, h]
}

#[derive(Serialize, Deserialize, Debug)]
pub struct DetectionFrame {
    pub timestamp: String,
    pub frame_number: u64,
    pub detections: Vec<Detection>,
    pub fps: f32,
    pub inference_time_ms: u64,
}

pub struct YOLODetector {
    pub class_names: Vec<String>,
}

impl YOLODetector {
    pub fn new(model_path: &str) -> Result<Self> {
        eprintln!("Initializing YOLO Detector with model: {}", model_path);
        
        let coco_classes = vec![
            "person", "bicycle", "car", "motorcycle", "airplane", "bus", "train", "truck", "boat", "traffic light",
            "fire hydrant", "stop sign", "parking meter", "bench", "bird", "cat", "dog", "horse", "sheep", "cow",
            "elephant", "bear", "zebra", "giraffe", "backpack", "umbrella", "handbag", "tie", "suitcase", "frisbee",
            "skis", "snowboard", "sports ball", "kite", "baseball bat", "baseball glove", "skateboard", "surfboard", "tennis racket", "bottle",
            "wine glass", "cup", "fork", "knife", "spoon", "bowl", "banana", "apple", "sandwich", "orange",
            "broccoli", "carrot", "hot dog", "pizza", "donut", "cake", "chair", "couch", "potted plant", "bed",
            "dining table", "toilet", "tv", "laptop", "mouse", "remote", "keyboard", "cell phone", "microwave", "oven",
            "toaster", "sink", "refrigerator", "book", "clock", "vase", "scissors", "teddy bear", "hair drier", "toothbrush"
        ];

        Ok(YOLODetector {
            class_names: coco_classes.into_iter().map(|s| s.to_string()).collect(),
        })
    }

    pub fn detect(&mut self, _frame: &Mat) -> Result<Vec<Detection>> {
        // Placeholder for now
        Ok(Vec::new())
    }
}

pub struct FaceDetector {
    cascade: objdetect::CascadeClassifier,
}

impl FaceDetector {
    pub fn new(cascade_path: &str) -> Result<Self> {
        let mut cascade = objdetect::CascadeClassifier::default()?;
        if !cascade.load(cascade_path)? {
            return Err(anyhow::anyhow!("Failed to load face cascade XML"));
        }
        Ok(FaceDetector { cascade })
    }

    pub fn detect(&mut self, frame: &Mat) -> Result<Vec<Detection>> {
        let mut gray = Mat::default();
        imgproc::cvt_color(frame, &mut gray, imgproc::COLOR_BGR2GRAY, 0)?;
        
        let mut faces = core::Vector::<core::Rect>::new();
        self.cascade.detect_multi_scale(
            &gray,
            &mut faces,
            1.1,
            3,
            objdetect::CASCADE_SCALE_IMAGE,
            core::Size::new(30, 30),
            core::Size::new(0, 0),
        )?;

        let mut detections = Vec::new();
        for face in faces {
            detections.push(Detection {
                class_id: -1, // -1 for face
                class_name: "face".to_string(),
                confidence: 1.0,
                bbox: [face.x, face.y, face.width, face.height],
            });
        }

        Ok(detections)
    }
}
