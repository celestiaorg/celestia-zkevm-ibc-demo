use blevm_prover::{BlockProver, BlockProverInput, CelestiaClient, CelestiaConfig, ProverConfig};
use celestia_types::nmt::Namespace;
use sp1_sdk::{include_elf, utils};
use std::{error::Error, fs};

#[tokio::main]
async fn main() -> Result<(), Box<dyn Error>> {
    // Setup logging.
    utils::setup_logger();
    // Load env variables.
    dotenv::dotenv().ok();
    // Throw errors if env variables are not set.
    std::env::var("CELESTIA_NODE_AUTH_TOKEN").expect("CELESTIA_NODE_AUTH_TOKEN must be set");
    std::env::var("CELESTIA_NAMESPACE").expect("CELESTIA_NAMESPACE must be set");

    // Initialize configurations
    let celestia_config = CelestiaConfig {
        node_url: "ws://localhost:26658".to_string(),
        auth_token: std::env::var("CELESTIA_NODE_AUTH_TOKEN")?,
    };

    let namespace = Namespace::new_v0(&hex::decode(std::env::var("CELESTIA_NAMESPACE")?)?)?;

    let prover_config = ProverConfig {
        elf_bytes: include_elf!("blevm"),
        // Uncomment the next line to generate a mock proof.
        // elf_bytes: include_elf!("blevm-mock"),
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

    // Generate proof
    println!("Generating proof...");
    let proof = prover.generate_proof(input).await?;
    // Save proof to file
    fs::write("proof.bin", proof)?;

    Ok(())
}
