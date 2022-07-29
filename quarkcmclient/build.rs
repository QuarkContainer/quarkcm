fn main() -> Result<(), Box<dyn std::error::Error>> {
    tonic_build::compile_protos("../pkg/connectionmanager/grpc/quarkcmsvc.proto")?;
    Ok(())
}