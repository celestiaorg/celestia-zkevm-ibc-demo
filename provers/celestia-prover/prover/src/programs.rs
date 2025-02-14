//! Programs for `sp1-ics07-tendermint`.

pub use sp1_ics07_tendermint_prover::programs::SP1Program;
use sp1_sdk::include_elf;

/// SP1 ICS07 Tendermint update client program.
pub struct UpdateClientProgram;

/// SP1 ICS07 Tendermint verify (non)membership program.
pub struct MembershipProgram;

impl SP1Program for UpdateClientProgram {
    const ELF: &[u8] = include_elf!("mock-membership");
}

impl SP1Program for MembershipProgram {
    const ELF: &[u8] = include_elf!("mock-update-client");
}
