fn main() -> Result<(), Box<dyn std::error::Error>> {
    tonic_build::configure()
        .build_server(true)
        .file_descriptor_set_path("proto_descriptor.bin")
        .compile_protos(&["../../proto/prover/v1/prover.proto"], &["../../proto"])?;
    Ok(())
}
