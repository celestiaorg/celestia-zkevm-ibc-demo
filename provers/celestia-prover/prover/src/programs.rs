//! Programs for `sp1-ics07-tendermint`.

pub use sp1_ics07_tendermint_prover::programs::SP1Program;
use sp1_sdk::include_elf;

/// SP1 ICS07 Tendermint update client program.
pub struct UpdateClientProgramFast;

/// SP1 ICS07 Tendermint verify (non)membership program.
pub struct MembershipProgramFast;

impl SP1Program for UpdateClientProgramFast {
    const ELF: &[u8] = include_elf!("membership-fast");
}

impl SP1Program for MembershipProgramFast {
    const ELF: &[u8] = include_elf!("update-client-fast");
}
