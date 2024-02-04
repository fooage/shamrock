use tonic_build;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Generate Protobuf Rust files based on IDL write in target directory.
    tonic_build::compile_protos("proto/block_service.proto")?;
    tonic_build::compile_protos("proto/meta_service.proto")?;

    Ok(())
}
