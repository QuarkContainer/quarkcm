# quarkcm

quarkcm is Connection Manager for Quark CDMA Networking Solution.

## push to docker hub
sudo docker login -u quarkcm

make clean
make

sudo docker image build -t quarkcm/quarkcm:v0.2.0 -f deploy/Dockerfile .
sudo docker image push quarkcm/quarkcm:v0.2.0
sudo docker image build -t quarkcm/quarkcni:v0.2.0 -f deploy/quarkcni.Dockerfile .
sudo docker image push quarkcm/quarkcni:v0.2.0

## deploy to cluster
kubectl apply -f deploy/deploy-quarkcm.yaml
