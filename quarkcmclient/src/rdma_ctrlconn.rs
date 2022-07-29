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

use spin::Mutex;
use std::collections::HashMap;

#[derive(Debug)]
pub struct CtrlInfo {
    // nodes: node ip --> Node
    pub nodes: Mutex<HashMap<u32, Node>>,

    // pods: pod ip --> Pod
    pub pods: Mutex<HashMap<u32, Pod>>,

    pub exiting: Mutex<bool>
}

impl Default for CtrlInfo {
    fn default() -> Self {
        return Self {
            nodes: Mutex::<HashMap<u32, Node>>::new(HashMap::<u32, Node>::new()),
            pods: Mutex::<HashMap<u32, Pod>>::new(HashMap::<u32, Pod>::new()),
            exiting: Mutex::<bool>::new(false),
        };
    }
}

#[derive(Debug)]
pub struct Node {
    pub name: String,
    pub hostname: String,
    pub ip: u32,
    pub timestamp: u64,
    pub resource_version: i32,
}

#[derive(Debug)]
pub struct Pod {
    pub key: String,
    pub ip: u32,
    pub node_name: String,
    pub resource_version: i32,
}
