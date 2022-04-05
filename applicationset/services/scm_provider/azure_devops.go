package scm_provider

import (
	"context"
	"fmt"

	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/core"
	azureGit "github.com/microsoft/azure-devops-go-api/azuredevops/git"
)

type AzureDevopsProvider struct {
	client       *core.Client
	connection   *azuredevops.Connection
	organization string
	teamProject  string
	accessToken  string
}

var _ SCMProviderService = &AzureDevopsProvider{}

func NewAzureDevopsProvider(ctx context.Context, accessToken string, org string, project string) (*AzureDevopsProvider, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("No access token provided")
	}

	url := fmt.Sprintf("https://dev.azure.com/%s", org)
	connection := azuredevops.NewPatConnection(url, accessToken)
	coreClient, err := core.NewClient(ctx, connection)
	if err != nil {
		return nil, err
	}

	return &AzureDevopsProvider{organization: org, teamProject: project, accessToken: accessToken, client: &coreClient, connection: connection}, nil
}

func (g *AzureDevopsProvider) ListRepos(ctx context.Context, cloneProtocol string) ([]*Repository, error) {
	gitClient, err := azureGit.NewClient(ctx, g.connection)
	if err != nil {
		return nil, err
	}
	getRepoArgs := azureGit.GetRepositoriesArgs{Project: &g.teamProject}
	azureRepos, err := gitClient.GetRepositories(ctx, getRepoArgs)

	if err != nil {
		return nil, err
	}
	repos := []*Repository{}
	for _, azureRepo := range *azureRepos {
		repos = append(repos, &Repository{
			Organization: g.organization,
			Repository:   *azureRepo.Name,
			URL:          *azureRepo.RemoteUrl,
			Branch:       *azureRepo.DefaultBranch,
			Labels:       []string{},
			RepositoryId: *&azureRepo.Id,
		})
	}

	return repos, nil
}

func (g *AzureDevopsProvider) RepoHasPath(ctx context.Context, repo *Repository, path string) (bool, error) {
	gitClient, err := azureGit.NewClient(ctx, g.connection)
	if err != nil {
		return false, err
	}

	idAsString := fmt.Sprintf("%v", repo.RepositoryId)
	branchName := repo.Branch
	getItemArgs := azureGit.GetItemArgs{RepositoryId: &idAsString, Project: &g.teamProject, Path: &path, VersionDescriptor: &azureGit.GitVersionDescriptor{Version: &branchName}}
	_, err = gitClient.GetItem(ctx, getItemArgs)

	if err != nil {
		wrappedError, isWrappedError := err.(azuredevops.WrappedError)
		if isWrappedError {
			if *wrappedError.TypeKey == "GitItemNotFoundException" {
				return false, nil
			}
		}
		return false, err
	}

	return true, nil
}

func (g *AzureDevopsProvider) GetBranches(ctx context.Context, repo *Repository) ([]*Repository, error) {
	gitClient, err := azureGit.NewClient(ctx, g.connection)
	if err != nil {
		return nil, err
	}

	getBranchesRequest := azureGit.GetBranchesArgs{RepositoryId: &repo.Repository, Project: &g.teamProject}
	branches, err := gitClient.GetBranches(ctx, getBranchesRequest)
	if err != nil {
		return nil, err
	}
	repos := []*Repository{}
	for _, azureBranch := range *branches {
		repos = append(repos, &Repository{
			Branch:       *azureBranch.Name,
			SHA:          *azureBranch.Commit.CommitId,
			Organization: repo.Organization,
			Repository:   repo.Repository,
			URL:          repo.URL,
			Labels:       []string{},
			RepositoryId: repo.RepositoryId,
		})
	}

	return repos, nil
}
