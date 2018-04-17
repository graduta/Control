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

package environment

import (
	"github.com/AliceO2Group/Control/common"
	"github.com/AliceO2Group/Control/configuration"
	"errors"
	"fmt"
	"strings"
	"encoding/json"
	"strconv"
)


type EnvironmentCfg struct {
	Roles           []RoleCfg           `json:"roles" binding:"required"`
}

type roleInfo struct {
	Name        string              `json:"name" binding:"required"`
	Command     *common.CommandInfo `json:"command" binding:"required"`
	WantsCPU    *float64            `json:"wantsCPU" binding:"required"`
	WantsMemory *float64            `json:"wantsMemory" binding:"required"`
	WantsPorts  Ranges              `json:"wantsPorts" binding:"required"`
	//Constraints []Constraint
}

/*
type Constraint struct {
	Attribute string
	Value     string //supports stuff like %hostname% etc.
	Operator  ConstraintOperator
}

type ConstraintOperator func(attribute string, value string) bool
*/

type Range struct {
	Begin uint64                        `json:"begin"`
	End   uint64                        `json:"end"`
}

type Ranges []Range

type RoleCfg struct {
	roleInfo
	RoleClass         string              `json:"roleClass" binding:"required"`
	CmdExtraEnv       []string            `json:"extraEnv,omitempty"`
	CmdExtraArguments []string            `json:"extraArgs,omitempty"`
}

func (this Ranges) Equals(other Ranges) (response bool) {
	if len(this) != len(other) {
		return false
	}

	response = true
	for i, _ := range this {
		if this[i].Begin == other[i].Begin && this[i].End == other[i].End {
			continue
		}
		response = false
		return
	}
	return
}

func (this RoleCfg) Equals(other *RoleCfg) (response bool) {
	return this.EqualsPtr(other)
}

func (this *RoleCfg) EqualsPtr(other *RoleCfg) (response bool) {
	if this == nil || other == nil {
		return false
	}

	if len(this.CmdExtraEnv) != len(other.CmdExtraEnv) ||
		len(this.CmdExtraArguments) != len(other.CmdExtraArguments) {
			return false
	}

	for i, _ := range this.CmdExtraEnv {
		if this.CmdExtraEnv[i] != other.CmdExtraEnv[i] {
			return false
		}
	}
	for i, _ := range this.CmdExtraArguments {
		if this.CmdExtraArguments[i] != other.CmdExtraArguments[i] {
			return false
		}
	}

	response = this.roleInfo.Equals(&other.roleInfo) &&
		       this.RoleClass == other.RoleClass
	return
}

func (this *roleInfo) Equals(other *roleInfo) (response bool) {
	if this == nil || other == nil {
		return false
	}
	response = this.Name == other.Name &&
		       this.Command.Equals(other.Command) &&
		       *this.WantsCPU == *other.WantsCPU &&
		       *this.WantsMemory == *other.WantsMemory &&
		       this.WantsPorts.Equals(other.WantsPorts)
	return
}

// mapToCmdInfo takes a configuration.Map with the correct contents and
// tries to generate the corresponding CommandInfo.
func mapToCmdInfo(iMap configuration.Map) (cmdInfo *common.CommandInfo, err error) {
	// Since the O² configuration mechanism only supports maps and strings
	// but not lists, we need to get the Map, JSON-unmarshal some strings
	// into slices and special-case a bool.
	// Then we JSON-marshal the Map and JSON-unmarshal it back into a fresh
	// CmdInfo instance.
	cmdInfo = &common.CommandInfo{}
	oMap := make(map[string]interface{}, 0)
	for k, v := range iMap {
		if k == "env" || k == "arguments" {
			sli := make([]string, 0, 0)
			err = json.Unmarshal([]byte(v.Value()), &sli)
			if err != nil {
				continue
			}
			oMap[k] = sli
		} else if k == "shell" {
			oMap[k] = v.Value() == "true"
		} else {
			oMap[k] = v.Value()
		}
	}

	marshaled, err := json.Marshal(oMap)

	err = json.Unmarshal(marshaled, cmdInfo)

	return
}

func roleCfgFromConfiguration(name string, cfgMap configuration.Map) (roleCfg *RoleCfg, err error)  {
	cfgErr := errors.New(fmt.Sprintf("bad configuration for role %s", name))

	ri, err := roleInfoFromConfiguration(name, cfgMap, false)
	if err != nil {
		return
	}

	roleClass := cfgMap["roleClass"]
	if roleClass == nil || !roleClass.IsValue() {	// roleClass is mandatory!
		err = cfgErr
		return
	}
	roleClassS := roleClass.Value()
	// ↑ so far so good, but we still don't know whether this string is a valid roleClass name

	cmdExtraEnv := cfgMap["cmdExtraEnv"]
	cmdExtraEnvSlice := make([]string, 0)
	if cmdExtraEnv == nil || !cmdExtraEnv.IsValue() {
		log.WithField("field", "cmdExtraEnv").
			Debug(cfgErr.Error())
	} else {
		cmdExtraEnvS := cmdExtraEnv.Value()
		// This string must be treated as a JSON list of strings, and unmarshaled into []string

		if strings.TrimSpace(cmdExtraEnvS) != "" {
			jErr := json.Unmarshal([]byte(cmdExtraEnvS), &cmdExtraEnvSlice)
			if jErr != nil {
				log.WithField("field", "cmdExtraEnv").
					Debug(cfgErr.Error())
				cmdExtraEnvSlice = []string{}
			}
		}
	}

	cmdExtraArguments := cfgMap["cmdExtraArguments"]
	cmdExtraArgumentsSlice := make([]string, 0)
	if cmdExtraArguments == nil || !cmdExtraArguments.IsValue() {
		log.WithField("field", "cmdExtraArguments").
			Debug(cfgErr.Error())
	} else {
		cmdExtraArgsS := cmdExtraArguments.Value()
		// This string must be treated as a JSON list of strings, and unmarshaled into []string

		if strings.TrimSpace(cmdExtraArgsS) != "" {
			jErr := json.Unmarshal([]byte(cmdExtraArgsS), &cmdExtraArgumentsSlice)
			if jErr != nil {
				log.WithField("field", "cmdExtraArguments").
					Debug(cfgErr.Error())
				cmdExtraArgumentsSlice = []string{}
			}
		}
	}

	roleCfg = &RoleCfg{
		roleInfo:          *ri,
		RoleClass:         roleClassS,
		CmdExtraEnv:       cmdExtraEnvSlice,
		CmdExtraArguments: cmdExtraArgumentsSlice,
	}
	return
}

func roleInfoFromConfiguration(name string, cfgMap configuration.Map, mandatoryFields bool) (ri *roleInfo, err error) {
	cfgErr := errors.New(fmt.Sprintf("bad configuration for roleClass %s", name))

	// Get WantsCPU
	wantsCPU := cfgMap["wantsCPU"]
	var wantsCPUF *float64 = nil
	if wantsCPU == nil || !wantsCPU.IsValue() || len(strings.TrimSpace(string(wantsCPU.Value()))) == 0 {
		if mandatoryFields {
			err = cfgErr
			return
		} else {
			log.WithField("field", "wantsCPU").
				Debug(cfgErr.Error())
		}
	} else {
		var val float64
		val, err = strconv.ParseFloat(string(wantsCPU.Value()), 64)
		if err != nil {
			err = errors.New(fmt.Sprintf("%s: %s",
				cfgErr.Error(), err.Error()))
			if mandatoryFields {
				return
			} else {
				log.WithField("field", "wantsCPU").
					Debug(err.Error())
				err = nil
			}
		} else {
			wantsCPUF = &val
		}
	}

	// Get WantsMemory
	wantsMemory := cfgMap["wantsMemory"]
	var wantsMemoryF *float64 = nil
	if wantsMemory == nil || !wantsMemory.IsValue() || len(strings.TrimSpace(string(wantsMemory.Value()))) == 0 {
		if mandatoryFields {
			err = cfgErr
			return
		} else {
			log.WithField("field", "wantsMemory").
				Debug(cfgErr.Error())
		}
	} else {
		var val float64
		val, err = strconv.ParseFloat(string(wantsMemory.Value()), 64)
		if err != nil {
			err = errors.New(fmt.Sprintf("%s: %s",
				cfgErr.Error(), err.Error()))
			if mandatoryFields {
				return
			} else {
				log.WithField("field", "wantsMemory").
					Debug(err.Error())
				err = nil
			}
		} else {
			wantsMemoryF = &val
		}
	}

	// Get the port ranges
	wantsPorts := cfgMap["wantsPorts"]
	var wantsPortsR Ranges = nil
	if wantsPorts == nil || !wantsPorts.IsValue() {
		if mandatoryFields {
			err = cfgErr
			return
		} else {
			log.WithField("field", "wantsPorts").
				Debug(cfgErr.Error())
		}
	} else {
		wantsPortsR, err = parsePortRanges(string(wantsPorts.Value()))
		if err != nil {
			err = errors.New(fmt.Sprintf("%s: %s",
				cfgErr.Error(), err.Error()))
			if mandatoryFields {
				return
			} else {
				log.WithField("field", "wantsPorts").
					Debug(err.Error())
				err = nil
			}
		}
	}

	// Get the CommandInfo
	var cmdInfo *common.CommandInfo = nil
	cmdInfoItem := cfgMap["command"]
	if cmdInfoItem == nil || !cmdInfoItem.IsMap() {
		if mandatoryFields {
			err = cfgErr
			return
		} else {
			log.WithField("field", "command").
				Debug(cfgErr.Error())
		}
	} else {
		cmdInfo, err = mapToCmdInfo(cmdInfoItem.Map())
		if err != nil {
			err = errors.New(fmt.Sprintf("%s: %s",
				cfgErr.Error(), err.Error()))
			if mandatoryFields {
				return
			} else {
				log.WithField("field", "command").
					Debug(err.Error())
				err = nil
			}
		}
	}

	ri = &roleInfo{
		Name:        name,
		Command:     cmdInfo,
		WantsCPU:    wantsCPUF,
		WantsMemory: wantsMemoryF,
		WantsPorts:  wantsPortsR,
	}
	return
}


// TODO: this is the FSM of each O² process, for further reference
//fsm := fsm.NewFSM(
//	"STANDBY",
//	fsm.Events{
//		{Name: "CONFIGURE", Src: []string{"STANDBY", "CONFIGURED"},           Dst: "CONFIGURED"},
//		{Name: "START",     Src: []string{"CONFIGURED"},                      Dst: "RUNNING"},
//		{Name: "STOP",      Src: []string{"RUNNING", "PAUSED"},               Dst: "CONFIGURED"},
//		{Name: "PAUSE",     Src: []string{"RUNNING"},                         Dst: "PAUSED"},
//		{Name: "RESUME",    Src: []string{"PAUSED"},                          Dst: "RUNNING"},
//		{Name: "EXIT",      Src: []string{"CONFIGURED", "STANDBY"},           Dst: "FINAL"},
//		{Name: "GO_ERROR",  Src: []string{"CONFIGURED", "RUNNING", "PAUSED"}, Dst: "ERROR"},
//		{Name: "RESET",     Src: []string{"ERROR"},                           Dst: "STANDBY"},
//	},
//	fsm.Callbacks{},
//)
//Fsm			*fsm.FSM		`json:"-"`	// skip
//			↑ this guy will initially only have 2 states: running and not running, or somesuch
//			  but he doesn't belong here because all this should be immutable



/*func NewO2Process() *ProcessDescriptor {
	return &ProcessDescriptor{
		Fsm: fsm.NewFSM(
			"STOPPED",
			fsm.Events{
				{Name: "START",	Src: []string{"STOPPED"},	Dst:"STARTED"},
				{Name: "STOP",	Src: []string{"STARTED"},	Dst:"STOPPED"},
			},
			fsm.Callbacks{},
		),
		Deployed: false,
	}
}*/