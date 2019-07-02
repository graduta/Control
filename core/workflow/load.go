/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018 CERN and copyright holders of ALICE O².
 * Author: Teo Mrnjavac <teo.mrnjavac@cern.ch>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * In applying this license CERN does not waive the privileges and
 * immunities granted to it by virtue of its status as an
 * Intergovernmental Organization or submit itself to any jurisdiction.
 */

package workflow

import (
	"github.com/AliceO2Group/Control/configuration"
	"github.com/AliceO2Group/Control/core/repos"
	"github.com/AliceO2Group/Control/core/task"
	"github.com/AliceO2Group/Control/core/the"

	"gopkg.in/yaml.v2"
	"io/ioutil"
)

// FIXME: workflowPath should be of type configuration.Path, not string
func Load(cfg configuration.ROSource, workflowPath string, parent Updatable, taskManager *task.Manager) (workflow Role, err error) {
	repoManager := the.RepoManager()

	var resolvedWorkflowPath string
	var workflowRepo *repos.Repo
	resolvedWorkflowPath, workflowRepo, err = repoManager.GetWorkflow(workflowPath) //Will fail if repo unknown
	if err != nil {
		return
	}

	var yamlDoc []byte
	yamlDoc, err = ioutil.ReadFile(resolvedWorkflowPath)
	if err != nil {
		return
	}

	root := new(aggregatorRole)
	root.parent = parent
	err = yaml.Unmarshal(yamlDoc, root)
	if err != nil {
		return nil, err
	}
	if parent != nil {
		root.parent = parent
	}

	workflow = root
	workflow.ProcessTemplates(workflowRepo)
	log.WithField("path", workflowPath).Debug("workflow loaded")
	//pp.Println(workflow)

	// Update class list
	taskClassesRequired := workflow.GetTaskClasses()
	err = repoManager.EnsureReposPresent(taskClassesRequired)
	if err != nil {
		return
	}
	err = taskManager.RefreshClasses(taskClassesRequired)
	return
}