use blevm_prover::{BlockProver, BlockProverInput, CelestiaClient, CelestiaConfig, ProverConfig};
use celestia_types::nmt::Namespace;
use sp1_sdk::{include_elf, utils};
use std::{error::Error, fs};
use std::time::Instant;

#[tokio::main]
async fn main() -> Result<(), Box<dyn Error>> {
    // Setup logging.
    utils::setup_logger();
    // Load env variables.
    dotenv::dotenv().ok();
    // Throw errors if env variables are not set.
    let auth_token =
        std::env::var("CELESTIA_NODE_AUTH_TOKEN").expect("CELESTIA_NODE_AUTH_TOKEN must be set");
    let namespace_hex =
        std::env::var("CELESTIA_NAMESPACE").expect("CELESTIA_NAMESPACE must be set");
    let node_url = std::env::var("CELESTIA_NODE_URL").expect("CELESTIA_NODE_URL must be set");

    let celestia_config = CelestiaConfig {
        node_url,
        auth_token,
    };

    let namespace = Namespace::new_v0(&hex::decode(namespace_hex)?)?;

    let prover_config = ProverConfig {
        // elf_bytes: include_elf!("blevm"),
        // Uncomment the next line to generate a mock proof.
        elf_bytes: include_elf!("blevm-mock"),
    };

    // Initialize the prover service
    let celestia_client = CelestiaClient::new(celestia_config, namespace).await?;
    let prover = BlockProver::new(celestia_client, prover_config);

    // Example input (replace with actual L2 block data)
    let input = BlockProverInput {
        // Hardcode the height of the block containing the blob
        // https://celenium.io/blob?commitment=eUbPUo7ddF77JSASRuZH1arKP7Ur8PYGtpW0qwvTP0w%3D&hash=AAAAAAAAAAAAAAAAAAAAAAAAAA8PDw8PDw8PDw8%3D&height=2988873
        block_height: 2988873,
        l2_block_data: fs::read("input/1/18884864.bin")?,
    };

    println!("Generating proof...");
    let start = Instant::now();
    let proof = prover.generate_proof(input).await?;
    let duration = start.elapsed();
    println!("Generated proof in {:?}.", duration);

    // Save proof to file
    println!("Saving proof to proof.bin");
    fs::write("proof.bin", proof)?;
    println!("Saved proof.");

    Ok(())
}
