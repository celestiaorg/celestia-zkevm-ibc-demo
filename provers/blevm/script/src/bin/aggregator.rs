use blevm_common::BlevmAggOutput;
use blevm_prover::prover::{
    AggregationInput, AggregatorConfig, BlockProver, CelestiaClient, CelestiaConfig, ProverConfig,
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
    inputs: Vec<String>,
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

    let prover = BlockProver::new(
        celestia_client,
        prover_config,
        aggregator_config,
        ProverClient::from_env(),
    );

    let mut aggregation_inputs: Vec<AggregationInput> = vec![];
    for input in args.inputs {
        let input_bin = fs::read(input)?;
        let aggregation_input = bincode::deserialize(&input_bin)?;
        aggregation_inputs.push(aggregation_input);
    }

    println!("aggregation_inputs.len(): {}", aggregation_inputs.len());

    if args.execute {
        println!("Executing...");
        let (public_values, execution_report) =
            prover.execute_aggregate_proofs(aggregation_inputs).await?;
        println!("Program executed successfully.");

        let output: BlevmAggOutput = bincode::deserialize(public_values.as_slice()).unwrap();
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
        let aggregation_output = prover.aggregate_proofs(aggregation_inputs).await?;
        let duration = start.elapsed();
        println!("Generated proof in {:?}.", duration);

        let proof_bin = bincode::serialize(&aggregation_output.proof)?;
        // Save proof to file
        println!("Saving proof to proof.bin");
        fs::write("proof.bin", proof_bin)?;
        println!("Saved proof.");
        return Ok(());
    }

    Ok(())
}
