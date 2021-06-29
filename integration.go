package main

type (
	Integration struct {
		ID         string  `json:"id"`
		Name       string  `json:"name"`
		Link       string  `json:"link"`
		Image      string  `json:"image"`
		Pipes      []*Pipe `json:"pipes"`
		AuthURL    string  `json:"auth_url,omitempty"`
		AuthType   string  `json:"auth_type,omitempty"`
		Authorized bool    `json:"authorized"`
		Deprecated bool    `json:"deprecated,omitempty"`
	}
)

func workspaceIntegrations(workspaceID int) ([]Integration, error) {
	authorizations, err := loadAuthorizations(workspaceID)
	if err != nil {
		return nil, err
	}
	workspacePipes, err := loadPipes(workspaceID)
	if err != nil {
		return nil, err
	}
	pipeStatuses, err := loadPipeStatuses(workspaceID)
	if err != nil {
		return nil, err
	}

	var integrations []Integration
	for j := range availableIntegrations {
		var integration = *availableIntegrations[j]
		integration.AuthURL = oAuth2URL(integration.ID)
		integration.Authorized = authorizations[integration.ID]
		var pipes []*Pipe
		pipesAreInUseForThisWS := false
		for i := range integration.Pipes {
			var pipe = *integration.Pipes[i]
			key := pipesKey(integration.ID, pipe.ID)

			existingPipe := workspacePipes[key]
			if existingPipe != nil {
				pipesAreInUseForThisWS = true
				pipe.Automatic = existingPipe.Automatic
				pipe.Configured = existingPipe.Configured
			}

			pipe.PipeStatus = pipeStatuses[key]
			pipes = append(pipes, &pipe)
		}
		if integration.Deprecated && !pipesAreInUseForThisWS {
			// Don't return this deprecated integration as available if
			// the workspace is not currently using it.
			continue
		}
		integration.Pipes = pipes
		integrations = append(integrations, integration)
	}
	return integrations, nil
}
