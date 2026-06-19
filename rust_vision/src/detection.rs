use opencv::{
    core::{self, Mat, MatTraitConst, Rect, Scalar, Size, Vector},
    dnn, imgproc, objdetect, prelude::*,
};
use serde::{Deserialize, Serialize};
use anyhow::Result;

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Detection {
    pub class_id: i32,
    pub class_name: String,
    pub confidence: f32,
    pub bbox: [i32; 4],
}

#[derive(Serialize, Deserialize, Debug)]
pub struct DetectionFrame {
    pub timestamp: String,
    pub frame_number: u64,
    pub detections: Vec<Detection>,
    pub fps: f32,
    pub inference_time_ms: u64,
}

const YOLO_INPUT_SIZE: i32 = 640;
const YOLO_CONFIDENCE_THRESHOLD: f32 = 0.25;
const YOLO_NMS_THRESHOLD: f32 = 0.45;
const YOLO_COLS: usize = 85;

pub struct YOLODetector {
    net: dnn::Net,
    class_names: Vec<String>,
}

impl YOLODetector {
    pub fn new(model_path: &str) -> Result<Self> {
        eprintln!("Initializing YOLO Detector with model: {}", model_path);
        let net = dnn::read_net_from_onnx(model_path)?;

        let class_names: Vec<String> = vec![
            "person".into(), "bicycle".into(), "car".into(), "motorcycle".into(),
            "airplane".into(), "bus".into(), "train".into(), "truck".into(),
            "boat".into(), "traffic light".into(), "fire hydrant".into(),
            "stop sign".into(), "parking meter".into(), "bench".into(),
            "bird".into(), "cat".into(), "dog".into(), "horse".into(),
            "sheep".into(), "cow".into(), "elephant".into(), "bear".into(),
            "zebra".into(), "giraffe".into(), "backpack".into(), "umbrella".into(),
            "handbag".into(), "tie".into(), "suitcase".into(), "frisbee".into(),
            "skis".into(), "snowboard".into(), "sports ball".into(), "kite".into(),
            "baseball bat".into(), "baseball glove".into(), "skateboard".into(),
            "surfboard".into(), "tennis racket".into(), "bottle".into(),
            "wine glass".into(), "cup".into(), "fork".into(), "knife".into(),
            "spoon".into(), "bowl".into(), "banana".into(), "apple".into(),
            "sandwich".into(), "orange".into(), "broccoli".into(), "carrot".into(),
            "hot dog".into(), "pizza".into(), "donut".into(), "cake".into(),
            "chair".into(), "couch".into(), "potted plant".into(), "bed".into(),
            "dining table".into(), "toilet".into(), "tv".into(), "laptop".into(),
            "mouse".into(), "remote".into(), "keyboard".into(), "cell phone".into(),
            "microwave".into(), "oven".into(), "toaster".into(), "sink".into(),
            "refrigerator".into(), "book".into(), "clock".into(), "vase".into(),
            "scissors".into(), "teddy bear".into(), "hair drier".into(), "toothbrush".into(),
        ];

        Ok(YOLODetector { net, class_names })
    }

    pub fn detect(&mut self, frame: &Mat) -> Result<Vec<Detection>> {
        let mut blob = dnn::blob_from_image(
            frame,
            1.0 / 255.0,
            Size::new(YOLO_INPUT_SIZE, YOLO_INPUT_SIZE),
            Scalar::new(0.0, 0.0, 0.0, 0.0),
            true,
            false,
            core::CV_32F,
        )?;

        self.net.set_input(&blob, "", 1.0, Scalar::default())?;

        let mut outputs = Vector::<Mat>::new();
        self.net.forward(&mut outputs, &Vector::<String>::new())?;

        if outputs.is_empty() {
            return Ok(Vec::new());
        }

        let frame_w = frame.cols();
        let frame_h = frame.rows();
        let scale_x = frame_w as f32 / YOLO_INPUT_SIZE as f32;
        let scale_y = frame_h as f32 / YOLO_INPUT_SIZE as f32;

        let mut candidates: Vec<(Rect, f32, i32)> = Vec::new();

        for i in 0..outputs.len() {
            let output = outputs.get(i)?;
            let total = output.total()? as usize;
            if total < YOLO_COLS {
                continue;
            }

            let data = output.data();
            if data.is_null() {
                continue;
            }
            let slice: &[f32] =
                unsafe { std::slice::from_raw_parts(data as *const f32, total) };

            let rows = total / YOLO_COLS;

            for row in 0..rows {
                let base = row * YOLO_COLS;
                let obj_conf = slice[base + 4];
                if obj_conf < YOLO_CONFIDENCE_THRESHOLD {
                    continue;
                }

                let mut max_score = 0.0f32;
                let mut class_id = -1i32;
                for c in 5..YOLO_COLS {
                    if slice[base + c] > max_score {
                        max_score = slice[base + c];
                        class_id = (c - 5) as i32;
                    }
                }

                let confidence = obj_conf * max_score;
                if confidence < YOLO_CONFIDENCE_THRESHOLD || class_id < 0 {
                    continue;
                }

                let cx = slice[base] * scale_x;
                let cy = slice[base + 1] * scale_y;
                let w = slice[base + 2] * scale_x;
                let h = slice[base + 3] * scale_y;

                let x = (cx - w / 2.0).max(0.0) as i32;
                let y = (cy - h / 2.0).max(0.0) as i32;
                let bw = w as i32;
                let bh = h as i32;

                candidates.push((Rect::new(x, y, bw, bh), confidence, class_id));
            }
        }

        if candidates.is_empty() {
            return Ok(Vec::new());
        }

        let mut boxes_vec = Vector::<Rect>::new();
        let mut scores_vec = Vector::<f32>::new();
        let mut class_ids_vec = Vector::<i32>::new();
        for (rect, score, cid) in &candidates {
            boxes_vec.push(*rect);
            scores_vec.push(*score);
            class_ids_vec.push(*cid);
        }

        let mut indices = Vector::<i32>::new();
        dnn::nms_boxes_batched_def(
            &boxes_vec,
            &scores_vec,
            &class_ids_vec,
            YOLO_CONFIDENCE_THRESHOLD,
            YOLO_NMS_THRESHOLD,
            &mut indices,
        )?;

        let mut detections = Vec::new();
        for i in 0..indices.len() {
            let idx = indices.get(i)? as usize;
            let (rect, confidence, class_id) = &candidates[idx];
            let class_name = if (*class_id as usize) < self.class_names.len() {
                self.class_names[*class_id as usize].clone()
            } else {
                format!("class_{}", class_id)
            };

            detections.push(Detection {
                class_id: *class_id,
                class_name,
                confidence: *confidence,
                bbox: [rect.x, rect.y, rect.width, rect.height],
            });
        }

        Ok(detections)
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

        let mut faces = Vector::<Rect>::new();
        self.cascade.detect_multi_scale(
            &gray,
            &mut faces,
            1.1,
            3,
            objdetect::CASCADE_SCALE_IMAGE,
            Size::new(30, 30),
            Size::new(0, 0),
        )?;

        let mut detections = Vec::new();
        for face in faces {
            detections.push(Detection {
                class_id: -1,
                class_name: "face".to_string(),
                confidence: 1.0,
                bbox: [face.x, face.y, face.width, face.height],
            });
        }

        Ok(detections)
    }
}
