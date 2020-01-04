package gitlab

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	gitlab "github.com/xanzy/go-gitlab"
)

func resourceGitlabProjectEnableRunner() *schema.Resource {
	return &schema.Resource{
		Create: resourceGitlabProjectEnableRunnerCreate,
		Read:   resourceGitlabProjectEnableRunnerRead,
		Delete: resourceGitlabProjectEnableRunnerDelete,
		// Importer: &schema.ResourceImporter{
		// 	State: resourceGitlabProjectEnableRunnerStateImporter,
		// },

		Schema: map[string]*schema.Schema{
			"runner_id": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"project_id": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceGitlabProjectEnableRunnerCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gitlab.Client)
	projectID := d.Get("project_id").(int)
	runnerID := d.Get("runner_id").(int)

	// https://godoc.org/github.com/xanzy/go-gitlab#EnableProjectRunnerOptions
	options := gitlab.EnableProjectRunnerOptions{
		RunnerID: runnerID,
	}

	log.Printf("[DEBUG] enable gitlab runner %v in project %v", runnerID, projectID)

	_, _, err := client.Runners.EnableProjectRunner(projectID, &options)
	if err != nil {
		return err
	}

	spid := fmt.Sprintf("%d", projectID)
	srid := fmt.Sprintf("%d", runnerID)
	d.SetId(buildTwoPartID(&spid, &srid))

	return resourceGitlabProjectEnableRunnerRead(d, meta)
}

func resourceGitlabProjectEnableRunnerRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gitlab.Client)

	projectID, runnerID, err := projectIDAndRunnerIDFromID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] checking gitlab runner %v is enabled in project %v", runnerID, projectID)

	options := gitlab.ListProjectRunnersOptions{}

	runners, _, err := client.Runners.ListProjectRunners(projectID, &options)
	if err != nil {
		return err
	}
	for _, runner := range runners {
		if runner.ID == runnerID {
			return nil
		}
	}

	// if we've reached here, we haven't seen our runner id in the project
	d.SetId("")
	return nil
}

func resourceGitlabProjectEnableRunnerDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gitlab.Client)

	projectID, runnerID, err := projectIDAndRunnerIDFromID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] disabling gitlab runner %v in project %v", runnerID, projectID)

	_, err = client.Runners.DisableProjectRunner(projectID, runnerID)

	return err
}

func projectIDAndRunnerIDFromID(id string) (int, int, error) {
	projectIDString, runnerIDString, err := parseTwoPartID(id)
	if err != nil {
		return 0, 0, err
	}

	projectID, err := strconv.Atoi(projectIDString)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get project: %v", err)
	}

	runnerID, err := strconv.Atoi(runnerIDString)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get runner: %v", err)
	}

	return projectID, runnerID, nil
}
