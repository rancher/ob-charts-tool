package branchverifycheck

// VerifyBranch takes a file path containing the ob-team-charts repo and verifies the branch state.
// It specifically is a tool to analyze the branch not the chart itself. They ensure consistency of process, not correctness of the changes.
//
// First it should verify basics: is path git repo, is upstream containing `ob-team-charts`, is on branch, is branch not main.
// The main verification steps should be:
// - Verify branch only modifies a single Package, (Only monitoring or Only logging; make this a soft-fail/warning)
// - Verify that the package+chart created is sequential to the last built chart for the package (this is hard fail)
// - If chart build scripts are run targeting the modified package no uncommitted changes are found (hard fail)
func VerifyBranch(path string) error {
	return nil
}
