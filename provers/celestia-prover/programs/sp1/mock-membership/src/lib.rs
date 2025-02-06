//! The crate that contains the types and utilities for `sp1-ics07-tendermint-membership` program.
#![deny(missing_docs, clippy::nursery, clippy::pedantic, warnings)]

use ibc_eureka_solidity_types::msgs::IMembershipMsgs::{KVPair, MembershipOutput};

/// The simplified function without the zkVM wrapper and without proof verification.
#[allow(clippy::missing_panics_doc)]
#[must_use]
pub fn membership(
    app_hash: [u8; 32],
    request_iter: impl Iterator<Item = (Vec<Vec<u8>>, Vec<u8>)>,
) -> MembershipOutput {
    let kv_pairs = request_iter
        .map(|(path, value)| KVPair {
            path: path.into_iter().map(Into::into).collect(),
            value: value.into(),
        })
        .collect();

    MembershipOutput {
        commitmentRoot: app_hash.into(),
        kvPairs: kv_pairs,
    }
}
