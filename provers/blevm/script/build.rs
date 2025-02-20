use sp1_build::build_program;

fn main() {
    build_program("../blevm");
    build_program("../blevm-mock");
    build_program("../blevm-aggregator");
}
