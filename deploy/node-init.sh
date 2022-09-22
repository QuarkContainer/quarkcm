#!/bin/bash

# Copyright 2022 quarkcm Authors.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

echo "ls"
ls 
echo "ls /var"
ls /var
cp -rf /var/quarkcni /home/
mkdir -p /etc/cni/net.d

nsenter -t 1 -m -u -n -i apt-get update -y && nsenter -t 1 -m -u -n -i apt-get install -y \
    sudo \
    rpcbind \
    rsyslog \
    libelf-dev \
    iproute2  \
    net-tools \
    iputils-ping \
    bridge-utils \
    ethtool \
    curl && \
nsenter -t 1 -m -u -n -i mkdir -p /opt/cni/bin && \
nsenter -t 1 -m -u -n -i mkdir -p /etc/cni/net.d && \
nsenter -t 1 -m -u -n -i cp -f /var/quarkcni/build/bin/quarkcni /opt/cni/bin/quarkcni && \
nsenter -t 1 -m -u -n -i cp -f /var/quarkcni/deploy/10-quarkcni.conf /etc/cni/net.d/10-0quarkcni.conf && \
echo "Quark Connection Manager init complete"