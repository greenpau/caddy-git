// Copyright 2022 Paul Greenberg greenpau@outlook.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package errors

// Config-related errors.
const (
	ErrRepositoryConfigNil                StandardError = "repository config is nil"
	ErrRepositoryConfigNameEmpty          StandardError = "repository config name is empty"
	ErrRepositoryConfigExists             StandardError = "repository config %q name already exists"
	ErrRepositoryConfigAddressEmpty       StandardError = "repository config address is empty"
	ErrRepositoryConfigAddressUnsupported StandardError = "repository config address %q is unsupported"
)
