use sp1_build::build_program;

fn main() {
    build_program_with_args("../blevm", Default::default());
    // build_program_with_args("../blevm-mock", Default::default());
}
