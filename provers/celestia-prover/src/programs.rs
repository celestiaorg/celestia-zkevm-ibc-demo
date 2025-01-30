//! Programs for `sp1-ics07-tendermint`.

// use sp1_sdk::{MockProver, Prover, SP1VerifyingKey};
pub use sp1_ics07_tendermint_prover::programs::SP1Program;
// use sp1_ics07_tendermint_prover::programs::{UpdateClientProgram, MembershipProgram};

/// SP1 ICS07 Tendermint update client program.
pub struct UpdateClientProgram;

/// SP1 ICS07 Tendermint verify (non)membership program.
pub struct MembershipProgram;

impl SP1Program for UpdateClientProgram {
    const ELF: &'static [u8] = include_bytes!("../../../elf/mock-update-client-elf");
}

impl SP1Program for MembershipProgram {
    const ELF: &'static [u8] = include_bytes!("../../../elf/mock-membership-elf");
}
