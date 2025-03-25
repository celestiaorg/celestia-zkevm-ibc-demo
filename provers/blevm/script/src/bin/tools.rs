use clap::Parser;
use rsp_client_executor::io::EthClientExecutorInput;
use std::{error::Error, fs};

// The arguments for the command.
#[derive(Parser, Debug)]
#[clap(author, version, about, long_about = None)]
struct Args {
    #[clap(long)]
    cmd: String,

    #[clap(long)]
    input: String,

    #[clap(long)]
    output: String,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn Error>> {
    // Load env variables.
    dotenv::dotenv().ok();

    // Parse the command line arguments.
    let args = Args::parse();

    if args.cmd == "dump-block" {
        let mut cache_file = std::fs::File::open(args.input)?;
        let client_input: EthClientExecutorInput = bincode::deserialize_from(&mut cache_file)?;
        let block_bytes = bincode::serialize(&client_input.current_block)?;
        fs::write(args.output, block_bytes)?;
    }
    Ok(())
}
