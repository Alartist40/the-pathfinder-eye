use opencv::{prelude::*, dnn, core};
use anyhow::Result;
use ndarray::Array1;

pub struct FaceRecognizer {
    net: dnn::Net,
}

impl FaceRecognizer {
    pub fn new(model_path: &str) -> Result<Self> {
        let net = dnn::read_net_from_onnx(model_path)?;
        Ok(FaceRecognizer { net })
    }

    pub fn extract_embedding(&mut self, face_image: &Mat) -> Result<Array1<f32>> {
        // FaceNet typically expects 160x160
        let blob = dnn::blob_from_image(
            face_image,
            1.0 / 127.5,
            core::Size::new(160, 160),
            core::Scalar::new(127.5, 127.5, 127.5, 0.0),
            true,
            false,
            core::CV_32F,
        )?;

        self.net.set_input(&blob, "", 1.0, core::Scalar::default())?;
        
        let mut output = Mat::default();
        self.net.forward(&mut output, &core::Vector::<String>::new())?;
        
        let mut embedding = Vec::new();
        // Assuming output is a 1D vector or has total elements
        let total = output.total();
        for i in 0..total {
            embedding.push(*output.at::<f32>(i as i32)?);
        }

        Ok(Array1::from(embedding))
    }

    pub fn compare_embeddings(emb1: &Array1<f32>, emb2: &Array1<f32>) -> f32 {
        let diff = emb1 - emb2;
        let dist = diff.dot(&diff).sqrt();
        // Return a similarity score where 1.0 is exact match
        1.0 / (1.0 + dist)
    }
}
