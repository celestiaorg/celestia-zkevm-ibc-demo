use bincode;
use celestia_types::consts::appconsts::LATEST_VERSION;
use celestia_types::AppVersion;
use celestia_types::{blob::Commitment, Blob, TxConfig};
use celestia_types::{
    nmt::{Namespace, NamespaceProof, NamespacedHashExt},
    ExtendedHeader,
};
use nmt_rs::simple_merkle::tree::MerkleHash;
use nmt_rs::{
    simple_merkle::{db::MemDb, proof::Proof, tree::MerkleTree},
    TmSha2Hasher,
};
use tendermint::{hash::Algorithm, Hash as TmHash};
use tendermint_proto::{
    v0_37::{types::BlockId as RawBlockId, version::Consensus as RawConsensusVersion},
    Protobuf,
};

use celestia_rpc::{BlobClient, Client, HeaderClient};
use core::cmp::max;
use rsp_client_executor::{
    io::ClientExecutorInput, ChainVariant, ClientExecutor, EthereumVariant, CHAIN_ID_ETH_MAINNET,
    CHAIN_ID_LINEA_MAINNET, CHAIN_ID_OP_MAINNET,
};
use sp1_sdk::{include_elf, ProverClient, SP1Stdin};
use std::fs;

/// The ELF (executable and linkable format) file for the Succinct RISC-V zkVM.
pub const BLEVM_ELF: &[u8] = include_elf!("blevm");

#[tokio::main]
async fn main() {
    dotenv::dotenv().ok();

    // Setup the client.
    let token = std::env::var("CELESTIA_NODE_AUTH_TOKEN").expect("Token not provided");
    let client = Client::new("ws://localhost:26658", Some(&token))
        .await
        .expect("Failed creating rpc client");

    // Use the namespace I posted the blob to
    let namespace: Namespace =
        Namespace::new_v0(&hex::decode("0f0f0f0f0f0f0f0f0f0f").unwrap()).unwrap();

    // Hardcode the height of the block containing the blob
    // https://celenium.io/blob?commitment=eUbPUo7ddF77JSASRuZH1arKP7Ur8PYGtpW0qwvTP0w%3D&hash=AAAAAAAAAAAAAAAAAAAAAAAAAA8PDw8PDw8PDw8%3D&height=2988873
    let height: u64 = 2988873;

    // Load the zkEVM input from a file, which contains the EVM block that becomes our blob
    let input_bytes = fs::read("input/1/18884864.bin").expect("could not read file");
    let input: ClientExecutorInput =
        bincode::deserialize(&input_bytes).expect("could not deserialize");

    // the EVM block from the input is our blob
    let block = input.current_block.clone();
    let block_bytes = bincode::serialize(&block).unwrap();
    let blob_from_file = Blob::new(namespace, block_bytes, AppVersion::V3).unwrap();
    println!(
        "commitment from test vector: {}",
        hex::encode(blob_from_file.commitment.0)
    );

    // Fetch the blob from the chain, so we can get its index (where it starts in the square)

    let blob_from_chain = client
        .blob_get(height, namespace, blob_from_file.commitment.clone())
        .await
        .expect("Failed getting blob");

    // Get the header and retrieve the EDS roots needed for proving inclusion
    let header: ExtendedHeader = client.header_get_by_height(height).await.unwrap();

    let eds_row_roots = header.dah.row_roots();
    let eds_column_roots = header.dah.column_roots();

    // Compute these values needed for proving inclusion
    let eds_size: u64 = eds_row_roots.len().try_into().unwrap();
    let ods_size = eds_size / 2;

    // Header hash is a merkle tree of the header fields
    // We can use this to prove the data hash is in the celestia header
    let hasher = TmSha2Hasher {};
    let mut header_field_tree: MerkleTree<MemDb<[u8; 32]>, TmSha2Hasher> =
        MerkleTree::with_hasher(hasher);

    let field_bytes = vec![
        Protobuf::<RawConsensusVersion>::encode_vec(header.header.version),
        header.header.chain_id.clone().encode_vec(),
        header.header.height.encode_vec(),
        header.header.time.encode_vec(),
        Protobuf::<RawBlockId>::encode_vec(header.header.last_block_id.unwrap_or_default()),
        header
            .header
            .last_commit_hash
            .unwrap_or_default()
            .encode_vec(),
        header.header.data_hash.unwrap_or_default().encode_vec(),
        header.header.validators_hash.encode_vec(),
        header.header.next_validators_hash.encode_vec(),
        header.header.consensus_hash.encode_vec(),
        header.header.app_hash.clone().encode_vec(),
        header
            .header
            .last_results_hash
            .unwrap_or_default()
            .encode_vec(),
        header.header.evidence_hash.unwrap_or_default().encode_vec(),
        header.header.proposer_address.encode_vec(),
    ];

    for leaf in field_bytes {
        header_field_tree.push_raw_leaf(&leaf);
    }

    let computed_header_hash = header_field_tree.root();
    let (data_hash_bytes_from_tree, data_hash_proof) = header_field_tree.get_index_with_proof(6);
    let data_hash_from_tree = TmHash::decode_vec(&data_hash_bytes_from_tree).unwrap();
    assert_eq!(
        data_hash_from_tree.as_bytes(),
        header.header.data_hash.unwrap().as_bytes()
    );
    assert_eq!(header.hash().as_ref(), header_field_tree.root());

    // Sanity check, verify the data hash merkle proof
    let hasher = TmSha2Hasher {};
    data_hash_proof
        .verify_range(
            &header_field_tree.root(),
            &[hasher.hash_leaf(&data_hash_bytes_from_tree)],
        )
        .unwrap();

    let nmt_multiproofs = client
        .blob_get_proof(height, namespace, blob_from_file.commitment.clone())
        .await
        .unwrap();
    /*
    let row_root_multiproof: Proof<TmSha2Hasher> =
        serde_json::from_str(&fs::read_to_string("row_root_multiproof.json").unwrap()).unwrap();
    println!(
        "row root multiproof len {:?}",
        row_root_multiproof.siblings().len()
    );*/

    let blob_index: u64 = blob_from_chain.index.unwrap();
    // calculate the blob_size, measured in "shares".
    // we do max(1, ...) as per the suggestion of Geometry team
    // need to double check if that's correct
    let blob_size: u64 = max(1, blob_from_chain.to_shares().unwrap().len() as u64);
    let first_row_index: u64 = blob_index.div_ceil(eds_size) - 1;
    let ods_index = blob_from_chain.index.unwrap() - (first_row_index * ods_size);

    let last_row_index: u64 = (ods_index + blob_size).div_ceil(ods_size) - 1;

    // Use TmSha2Hasher to merklize the row and column roots, then compute a range proof of the row roots spanned by the blob
    let hasher = TmSha2Hasher {};
    let mut row_root_tree: MerkleTree<MemDb<[u8; 32]>, TmSha2Hasher> =
        MerkleTree::with_hasher(hasher);

    let leaves = eds_row_roots
        .iter()
        .chain(eds_column_roots.iter())
        .map(|root| root.to_array())
        .collect::<Vec<[u8; 90]>>();

    for root in &leaves {
        row_root_tree.push_raw_leaf(root);
    }

    // assert that the row root tree equals the data hash
    assert_eq!(row_root_tree.root(), data_hash_from_tree.as_bytes());
    // Get range proof of the row roots spanned by the blob
    // +1 is so we include the last row root
    let row_root_multiproof =
        row_root_tree.build_range_proof(first_row_index as usize..(last_row_index + 1) as usize);
    // Sanity check, verify the row root range proof
    let hasher = TmSha2Hasher {};
    let leaves_hashed = leaves
        .iter()
        .map(|leaf| hasher.hash_leaf(leaf))
        .collect::<Vec<[u8; 32]>>();
    row_root_multiproof
        .verify_range(
            data_hash_from_tree.as_bytes().try_into().unwrap(),
            &leaves_hashed[first_row_index as usize..(last_row_index + 1) as usize],
        )
        .unwrap();

    // Setup the logger.
    sp1_sdk::utils::setup_logger();

    // Setup the prover client.
    let client = ProverClient::new();

    // Setup the inputs.
    let mut stdin = SP1Stdin::new();
    stdin.write(&input);
    stdin.write(&namespace);
    stdin.write(&header.header.hash());
    stdin.write_vec(data_hash_bytes_from_tree);
    stdin.write(&data_hash_proof);
    stdin.write(&row_root_multiproof);
    stdin.write(&nmt_multiproofs);
    stdin.write(&eds_row_roots[first_row_index as usize..(last_row_index + 1) as usize].to_vec());

    // Serialize stdin to file for debugging
    let stdin_bytes = bincode::serialize(&stdin).expect("Failed to serialize stdin");
    fs::write("stdin.bin", stdin_bytes).expect("Failed to write stdin to file");

    //let (output, report) = client.execute(BLEVM_ELF, stdin).run().unwrap();

    let (pk, vk) = client.setup(&BLEVM_ELF);

    println!("Generating proof...");
    let proof = client.prove(&pk, stdin).core().run().unwrap();

    let proof_bytes = bincode::serialize(&proof).expect("Failed to serialize proof");
    fs::write("proof.bin", proof_bytes).expect("Failed to write proof to file");
}
