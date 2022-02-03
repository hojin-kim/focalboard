package app

import (
	"errors"
	"github.com/mattermost/focalboard/server/model"
	"github.com/mattermost/focalboard/server/services/store"
)

const (
	KeyPrefix                 = "focalboard_" // use key prefix to namespace focalboard props
	KeyOnboardingTourStarted  = KeyPrefix + "onboardingTourStarted"
	KeyOnboardingTourCategory = KeyPrefix + "tourCategory"
	KeyOnboardingTourStep     = KeyPrefix + "onboardingTourStep"
	KeyOnboardingTourSkipped  = KeyPrefix + "onboardingTourSkipped"

	ValueOnboardingFirstStep    = "0"
	ValueTourCategoryOnboarding = "onboarding"

	// OnboardingBoardID is the board ID from template.json.
	// TODO make this more durable
	OnboardingBoardID = "buixxjic3xjfkieees4iafdrznc"

	WelcomeBoardTitle = "Welcome to Boards!"
)

func (a *App) PrepareOnboardingTour(userID string) (string, string, error) {
	// create a private workspace for the user
	workspaceID, err := a.store.CreatePrivateWorkspace(userID)
	if err != nil {
		return "", "", err
	}

	// copy the welcome board into this workspace
	boardID, err := a.createWelcomeBoard(userID, workspaceID)
	if err != nil {
		return "", "", err
	}

	// set user's tour state to initial state
	userPropPatch := model.UserPropPatch{
		UpdatedFields: map[string]interface{}{
			KeyOnboardingTourStarted:  true,
			KeyOnboardingTourStep:     ValueOnboardingFirstStep,
			KeyOnboardingTourCategory: ValueTourCategoryOnboarding,
		},
	}
	if err := a.store.PatchUserProps(userID, userPropPatch); err != nil {
		return "", "", err
	}

	return workspaceID, boardID, nil
}

func (a *App) createWelcomeBoard(userID, workspaceID string) (string, error) {
	blocks, err := a.GetSubTree(store.Container{WorkspaceID: "0"}, OnboardingBoardID, 3)
	if err != nil {
		return "", err
	}

	blocks = model.GenerateBlockIDs(blocks, a.logger)

	// we're copying from a global template, so we need to set the
	// isTemplate flag to false on the board
	var welcomeBoardID string
	for i := range blocks {
		if blocks[i].Type == model.TypeBoard {
			blocks[i].Fields["isTemplate"] = false
		}

		if blocks[i].Title == WelcomeBoardTitle {
			welcomeBoardID = blocks[i].ID
			break
		}
	}

	model.StampModificationMetadata(userID, blocks, nil)
	_, err = a.InsertBlocks(store.Container{WorkspaceID: workspaceID}, blocks, userID, false)
	if err != nil {
		return "", err
	}

	if welcomeBoardID == "" {
		return "", errors.New("unable to find welcome board in newly created blocks")
	}

	return welcomeBoardID, nil
}