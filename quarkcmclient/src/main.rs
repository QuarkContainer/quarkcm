/*
Copyright 2022 quarkcm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

pub mod constants;
pub mod rdma_ctrlconn;

mod svc_client;
use crate::rdma_ctrlconn::*;
use svc_client::node_informer::NodeInformer;
use svc_client::pod_informer::PodInformer;

use lazy_static::lazy_static;

lazy_static! {
    pub static ref RDMA_CTLINFO: CtrlInfo = CtrlInfo::default();
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let mut pod_informer = PodInformer::new().await?;
    let mut node_informer = NodeInformer::new().await?;

    let (pod_informer_result, node_informer_result) =
        tokio::join!(pod_informer.run(), node_informer.run(),);
    match pod_informer_result {
        Err(e) => println!("Pod informer error: {:?}", e),
        _ => (),
    }
    match node_informer_result {
        Err(e) => println!("Node informer error: {:?}", e),
        _ => (),
    }
    println!("Exit!");
    Ok(())
}
