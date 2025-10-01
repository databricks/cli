Guidelines on writing a resource

 - See adapter.go on what methods are needed and what constraints are present.
 - Return SDK errors directly, no need to wrap it. Things like current operation, resource key, id are already part of the error message.
 - Although the arguments are pointers, they are never nil, so nil checks are not needed.
 - The arguments point to actual struct that will be persisted in state, any changes to it will affect what is stored in state. Usually there is no need to change it, but if there is, there should always be detailed explanation.
 - Each Create/Update/Delete method should correspond to one API call. We persist state right after, so there is minimum chance of having orphaned resources.
 - The logic what kind of update it is should be in FieldTriggers / ClassifyChange methods. The methods performing update should not have logic in them on what method to call.
 - Create/Update/Delete methods should not need to read any state. (We can implement support for passing remoteState we already to these methods if such need arises though).
 - Prefer “with refresh” variants of CRU methods if API supports that. That avoids explicit DoRefresh() call.

Nice to have
 - Add link to corresponding API documentation before each method.
 - Add link to corresponding terraform resource implementation at the top of the file.
