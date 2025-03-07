//! This script generates a blevm or blevm-mock proof.
//!
//! You can run this script using the following command:
//! ```shell
//! RUST_LOG=info cargo run --release -- --execute
//! ```
//! or
//! ```shell
//! RUST_LOG=info cargo run --release -- --prove
//! ```
//! and you can use blevm-mock in execute or prove mode:
//! ```shell
//! RUST_LOG=info cargo run --release -- --prove --mock
//! ```
use blevm_common::BlevmOutput;
use blevm_prover::{
    AggregatorConfig, BlockProver, BlockProverInput, CelestiaClient, CelestiaConfig, ProverConfig,
};
use celestia_types::nmt::Namespace;
use clap::Parser;
use sp1_sdk::{include_elf, utils};
use std::time::Instant;
use std::{error::Error, fs};

pub const BLEVM_ELF: &[u8] = include_elf!("blevm");
pub const BLEVM_MOCK_ELF: &[u8] = include_elf!("blevm-mock");
pub const BLEVM_AGGREGATOR_ELF: &[u8] = include_elf!("blevm-aggregator");

// The arguments for the command.
#[derive(Parser, Debug)]
#[clap(author, version, about, long_about = None)]
struct Args {
    #[clap(long)]
    execute: bool,

    #[clap(long)]
    prove: bool,

    #[clap(long)]
    mock: bool,

    #[clap(long)]
    aggregate: bool,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn Error>> {
    // Setup logging.
    utils::setup_logger();
    // Load env variables.
    dotenv::dotenv().ok();

    // Parse the command line arguments.
    let args = Args::parse();

    if args.execute == args.prove {
        eprintln!("Error: You must specify either --execute or --prove");
        std::process::exit(1);
    }

    if args.mock {
        println!("In mock mode so using BLEVM_MOCK_ELF.")
    } else {
        println!("Not in mock mode so using BLEVM_ELF.")
    }
    let prover_config = ProverConfig {
        elf_bytes: if args.mock { BLEVM_MOCK_ELF } else { BLEVM_ELF },
    };

    let celestia_config = CelestiaConfig {
        node_url: std::env::var("CELESTIA_NODE_URL").expect("CELESTIA_NODE_URL must be set"),
        auth_token: std::env::var("CELESTIA_NODE_AUTH_TOKEN")
            .expect("CELESTIA_NODE_AUTH_TOKEN must be set"),
    };
    let namespace_hex =
        std::env::var("CELESTIA_NAMESPACE").expect("CELESTIA_NAMESPACE must be set");
    let namespace = Namespace::new_v0(&hex::decode(namespace_hex)?)?;
    let celestia_client = CelestiaClient::new(celestia_config, namespace).await?;

    let aggregator_config = AggregatorConfig {
        elf_bytes: BLEVM_AGGREGATOR_ELF,
    };
    let prover = BlockProver::new(celestia_client, prover_config, aggregator_config);

    // Example input (replace with actual L2 block data)
    let input = BlockProverInput {
        // Hardcode the height of the block containing the blob
        // https://celenium.io/blob?commitment=eUbPUo7ddF77JSASRuZH1arKP7Ur8PYGtpW0qwvTP0w%3D&hash=AAAAAAAAAAAAAAAAAAAAAAAAAA8PDw8PDw8PDw8%3D&height=2988873
        block_height: 2988873,
        l2_block_data: fs::read("input/blevm/1/18884864.bin").expect(
            "Failed to load L2 block data. Ensure you're in a directory with input/blevm/1/18884864.bin",
        ),
    };

    if args.execute {
        println!("Executing...");
        let (public_values, execution_report) = prover.execute_generate_proof(input).await?;
        println!("Program executed successfully.");

        let output: BlevmOutput = bincode::deserialize(public_values.as_slice()).unwrap();
        println!("{:?}", output);
        println!(
            "Number of cycles: {}",
            execution_report.total_instruction_count()
        );
        return Ok(());
    }

    if args.prove {
        println!("Generating proof...");
        let start = Instant::now();
        let (proof, _) = prover.generate_proof(input).await?;
        let duration = start.elapsed();
        println!("Generated proof in {:?}.", duration);

        let proof_bin = bincode::serialize(&proof)?;
        // Save proof to file
        println!("Saving proof to proof.bin");
        fs::write("proof.bin", proof_bin)?;
        println!("Saved proof.");
        return Ok(());
    }

    Ok(())
}
