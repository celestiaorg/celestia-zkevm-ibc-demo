use blevm_common::{BlockProver, BlockProverInput, CelestiaClient, CelestiaConfig, ProverConfig};
use celestia_types::nmt::Namespace;
use sp1_sdk::include_elf;
use std::{error::Error, fs};

#[tokio::main]
async fn main() -> Result<(), Box<dyn Error>> {
    // Initialize configurations
    let celestia_config = CelestiaConfig {
        node_url: "ws://localhost:26658".to_string(),
        auth_token: std::env::var("CELESTIA_NODE_AUTH_TOKEN")?,
    };

    let namespace = Namespace::new_v0(&hex::decode(std::env::var("CELESTIA_NAMESPACE")?)?)?;

    let prover_config = ProverConfig {
        elf_bytes: include_elf!("blevm"),
    };

    // Initialize the prover service
    let celestia_client = CelestiaClient::new(celestia_config, namespace).await?;
    let prover = BlockProver::new(celestia_client, prover_config);

    // Example input (replace with actual L2 block data)
    let input = BlockProverInput {
        block_height: 2988873,
        l2_block_data: fs::read("input/1/18884864.bin")?,
    };

    // Generate proof
    let proof = prover.generate_proof(input).await?;

    // Save proof to file
    fs::write("proof.bin", proof)?;

    Ok(())
}
