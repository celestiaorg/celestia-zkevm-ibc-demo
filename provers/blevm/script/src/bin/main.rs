//! This script generates a blevm proof.
//!
//! You can run this script using the following command:
//! ```shell
//! RUST_LOG=info cargo run --release -- --execute
//! ```
//! or
//! ```shell
//! RUST_LOG=info cargo run --release -- --prove
//! ```
use blevm_common::BlevmOutput;
use blevm_prover::prover::{
    AggregationInput, AggregatorConfig, BlockProver, BlockProverInput, CelestiaClient,
    CelestiaConfig, ProverConfig,
};
use celestia_types::nmt::Namespace;
use clap::Parser;
use sp1_sdk::{include_elf, ProverClient, utils};
use std::time::Instant;
use std::{error::Error, fs};

pub const BLEVM_ELF: &[u8] = include_elf!("blevm");
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
    input_path: String,

    #[clap(long)]
    rollup_block_path: String,

    #[clap(long)]
    inclusion_block: u64,
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

    let prover_config = ProverConfig {
        elf_bytes: BLEVM_ELF,
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
    let sp1_client = ProverClient::from_env();
    let (_, aggregator_vkey) = sp1_client.setup(BLEVM_AGGREGATOR_ELF);

    let prover = BlockProver::new(celestia_client, prover_config, aggregator_config, sp1_client);

    let input = BlockProverInput {
        inclusion_height: args.inclusion_block,
        client_executor_input: fs::read(args.input_path).expect("failed to read input file"),
        rollup_block: fs::read(args.rollup_block_path).expect("failed to read rollup block file"),
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
        let (proof, vk) = prover.generate_proof(input).await?;
        let duration = start.elapsed();
        println!("Generated proof in {:?}.", duration);

        let aggregation_input = AggregationInput { proof, vk };
        let aggregation_input_bin = bincode::serialize(&aggregation_input)?;

        println!("Saving proof to proof.bin.");
        fs::write("proof.bin", aggregation_input_bin)?;
        println!("Saved proof.");

        return Ok(());
    }

    Ok(())
}
