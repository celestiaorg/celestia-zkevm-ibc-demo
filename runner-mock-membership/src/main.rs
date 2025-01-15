use sp1_sdk::{ProverClient, SP1Stdin};

pub const MOCK_MEMBERSHIP_ELF: &[u8] = include_bytes!("../../target/elf-compilation/riscv32im-succinct-zkvm-elf/release/mock-membership");

fn main() {
    sp1_sdk::utils::setup_logger();
    let client = ProverClient::new();

    let mut stdin = SP1Stdin::new();
    let vec1 = vec![0; 32];
    let request_len = vec![1u8];
    let path1 = vec![0u8; 32];
    let value1 = vec![0u8; 32];
    stdin.write_vec(vec1);
    stdin.write_vec(request_len);
    stdin.write_vec(path1);
    stdin.write_vec(value1);

    //client.execute(&MOCK_MEMBERSHIP_ELF, stdin).run().unwrap();
    let (pk, vk) = client.setup(&MOCK_MEMBERSHIP_ELF);
    let start = std::time::Instant::now();
    let proof = client.prove(&pk, stdin.clone()).core().run().unwrap();
    println!("Core Stark Proving took: {:?}", start.elapsed());

    let start = std::time::Instant::now();
    let proof = client.prove(&pk, stdin.clone()).compressed().run().unwrap();
    println!("Compressed Proving took: {:?}", start.elapsed());

    let start = std::time::Instant::now();
    let proof = client.prove(&pk, stdin.clone()).groth16().run().unwrap();
    println!("Groth16 Proving took: {:?}", start.elapsed());
}
