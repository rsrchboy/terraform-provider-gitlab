package gitlab

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	gitlab "github.com/xanzy/go-gitlab"
)

func resourceGitlabRunner() *schema.Resource {
	return &schema.Resource{
		Create: resourceGitlabRunnerCreate,
		Read:   resourceGitlabRunnerRead,
		Update: resourceGitlabRunnerUpdate,
		Delete: resourceGitlabRunnerDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"registration_token": {
				Type:      schema.TypeString,
				ForceNew:  true,
				Required:  true,
				Sensitive: true,
			},
			"token": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"runner_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"access_level": {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"not_protected", "ref_protected"}, true),
			},
			"revision": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"locked": {
				Type:     schema.TypeBool,
				Computed: true,
				Optional: true,
			},
			"is_shared": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"maximum_timeout": {
				Type:         schema.TypeInt,
				Computed:     true,
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(10 * 60),
			},
			"tags": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"active": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"run_untagged": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"ip_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"contacted_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"online": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"architecture": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"projects": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name_with_namespace": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"path": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"path_with_namespace": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"groups": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"web_url": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func resourceGitlabRunnerCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gitlab.Client)

	// https://godoc.org/github.com/xanzy/go-gitlab#RegisterNewRunnerOptions
	options := gitlab.RegisterNewRunnerOptions{
		Token:       gitlab.String(d.Get("registration_token").(string)),
		Description: gitlab.String(d.Get("description").(string)),
		RunUntagged: gitlab.Bool(d.Get("run_untagged").(bool)),
		Active:      gitlab.Bool(d.Get("active").(bool)),
		Locked:      gitlab.Bool(d.Get("locked").(bool)),
	}

	if v, ok := d.GetOk("tags"); ok {
		options.TagList = *(stringSetToStringSlice(v.(*schema.Set)))
	}

	if v, ok := d.GetOk("maximum_timeout"); ok {
		options.MaximumTimeout = gitlab.Int(v.(int))
	}

	log.Printf("[DEBUG] create gitlab runner")

	runnerDetails, _, err := client.Runners.RegisterNewRunner(&options)
	if err != nil {
		return err
	}

	d.SetId(fmt.Sprintf("%d", runnerDetails.ID))

	// return resourceGitlabRunnerRead(d, meta)
	// some options, like access_level, are either not supported on
	// register or not supported by go-gitlab on register
	return resourceGitlabRunnerUpdate(d, meta)
}

func resourceGitlabRunnerRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gitlab.Client)

	runnerID, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	// FIXME probably ought to VerifyRegisteredRunner() here first

	log.Printf("[DEBUG] read gitlab runner %d", runnerID)

	v, _, err := client.Runners.GetRunnerDetails(runnerID)
	if err != nil {
		return err
	}

	d.Set("runner_id", v.ID)
	d.Set("token", v.Token)
	d.Set("description", v.Description)
	d.Set("access_level", v.AccessLevel)
	d.Set("revision", v.Revision)
	d.Set("version", v.Version)
	d.Set("is_shared", v.IsShared)
	d.Set("maximum_timeout", v.MaximumTimeout)
	d.Set("tags", v.TagList)
	d.Set("locked", v.Locked)
	d.Set("online", v.Online)
	d.Set("status", v.Status)
	d.Set("ip_address", v.IPAddress)
	d.Set("contacted_at", v.ContactedAt)
	d.Set("architecture", v.Architecture)
	d.Set("name", v.Name)
	// d.Set("X", v.X)

	projectsList := []interface{}{}
	for _, project := range v.Projects {
		log.Printf("[DEBUG] read gitlab runner %d project %d", runnerID, project.ID)
		values := map[string]interface{}{
			"id":                  project.ID,
			"name":                project.Name,
			"name_with_namespace": project.NameWithNamespace,
			"path":                project.Path,
			"path_with_namespace": project.PathWithNamespace,
		}
		projectsList = append(projectsList, values)
	}
	d.Set("projects", projectsList)

	groupsList := []interface{}{}
	for _, group := range v.Groups {
		log.Printf("[DEBUG] read gitlab runner %d group %d", runnerID, group.ID)
		values := map[string]interface{}{
			"id":      group.ID,
			"name":    group.Name,
			"web_url": group.WebURL,
		}
		groupsList = append(groupsList, values)
	}
	d.Set("groups", groupsList)

	return nil
}

func resourceGitlabRunnerUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gitlab.Client)
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	// https://godoc.org/github.com/xanzy/go-gitlab#UpdateRunnerDetailsOptions
	options := gitlab.UpdateRunnerDetailsOptions{
		Description: gitlab.String(d.Get("description").(string)),
		RunUntagged: gitlab.Bool(d.Get("run_untagged").(bool)),
		Active:      gitlab.Bool(d.Get("active").(bool)),
		Locked:      gitlab.Bool(d.Get("locked").(bool)),
		AccessLevel: gitlab.String(d.Get("access_level").(string)),
		// MaximumTimeout: gitlab.Int(d.Get("maximum_timeout").(int)),
		// X: gitlab.String(d.Get("X").(string)),
	}

	if v, ok := d.GetOk("tags"); ok {
		options.TagList = *(stringSetToStringSlice(v.(*schema.Set)))
	}

	if v, ok := d.GetOk("maximum_timeout"); ok {
		options.MaximumTimeout = gitlab.Int(v.(int))
	}

	log.Printf("[DEBUG] update gitlab runner %d", id)

	_, _, err = client.Runners.UpdateRunnerDetails(id, &options)
	if err != nil {
		return err
	}

	return resourceGitlabRunnerRead(d, meta)
}

func resourceGitlabRunnerDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gitlab.Client)
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Delete gitlab runner %d", id)

	_, err = client.Runners.RemoveRunner(id)
	return err
}
