use sp1_helper::build_program_with_args;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    tonic_build::configure()
        .build_server(true)
        .file_descriptor_set_path("proto_descriptor.bin")
        .compile_protos(
            &[
                "../../proto/prover/v1/prover.proto",
                "../../proto/ibc/lightclients/groth16/v1/groth16.proto",
            ],
            &["../../proto"],
        )?;
    build_program_with_args("../blevm/blevm", Default::default());
    build_program_with_args("../blevm/blevm-mock", Default::default());
    build_program_with_args("../blevm/blevm-aggregator", Default::default());
    Ok(())
}
