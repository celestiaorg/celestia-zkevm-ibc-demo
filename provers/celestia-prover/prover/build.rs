use sp1_build::build_program;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    tonic_build::configure()
        .build_server(true)
        .file_descriptor_set_path("proto_descriptor.bin")
        .compile_protos(
            &["../../../proto/prover/v1/prover.proto"],
            &["../../../proto"],
        )?;
    build_program("../programs/sp1/mock-membership");
    build_program("../programs/sp1/mock-update-client");
    Ok(())
}
