//! A mock program that uses the same inputs to verify tendermint headers

// These two lines are necessary for the program to properly compile.
//
// Under the hood, we wrap your main function with some extra code so that it behaves properly
// inside the zkVM.
#![no_main]
sp1_zkvm::entrypoint!(main);

use alloy_sol_types::SolValue;
use ibc_client_tendermint_types::{ConsensusState, Header};
use ibc_eureka_solidity_types::msgs::IICS07TendermintMsgs::{
    ClientState as SolClientState, ConsensusState as SolConsensusState,
};
use ibc_eureka_solidity_types::msgs::IUpdateClientMsgs::UpdateClientOutput;

/// The main function of the program.
///
/// # Panics
/// Panics if the verification fails.
pub fn main() {
    let encoded_1 = sp1_zkvm::io::read_vec();
    let encoded_2 = sp1_zkvm::io::read_vec();
    let encoded_3 = sp1_zkvm::io::read_vec();
    let encoded_4 = sp1_zkvm::io::read_vec();

    // input 1: the client state
    // let client_state = bincode::deserialize::<SolClientState>(&encoded_1).unwrap();
    let client_state = SolClientState::abi_decode(&encoded_1, true).unwrap();

    // input 2: the trusted consensus state
    let trusted_consensus_state: ConsensusState = SolConsensusState::abi_decode(&encoded_2, true)
        .unwrap()
        .into();
    // input 3: the proposed header
    let proposed_header = serde_cbor::from_slice::<Header>(&encoded_3).unwrap();
    // input 4: time
    let time = u64::from_le_bytes(encoded_4.try_into().unwrap());

    let trusted_height = proposed_header.trusted_height.try_into().unwrap();
    let new_height = proposed_header.height().try_into().unwrap();
    let new_consensus_state = ConsensusState::from(proposed_header);

    // output: generate the public output
    let output = UpdateClientOutput {
        clientState: client_state,
        trustedConsensusState: trusted_consensus_state.into(),
        newConsensusState: new_consensus_state.into(),
        time,
        trustedHeight: trusted_height,
        newHeight: new_height,
    };

    sp1_zkvm::io::commit_slice(&output.abi_encode());
}
