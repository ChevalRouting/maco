// @x-maco {"version":1,"instructions":"Start with currentUser to check permissions, hostInfo for capacity, and listVMs for VM IDs and state. Before creation, discover image keys and provisioning support with listCatalog, downloaded images with listImages, network IDs with listNetworks, and media IDs with listMedia. Use operation request schemas, not stored manifests, for action bodies. Memory uses MiB and disk capacity uses GiB. A 202 response is a queued job: use its id with waitJob or getJob, and inspect state, result and error. Only succeeded confirms completion. Do not repeat a mutation merely because waiting disconnected. Use createBackup then waitJob and listBackups to verify a backup; use restoreBackup with as_new=true to preserve the original VM. Prefer shutdownVM over forced stopVM. Host power actions interrupt the connection. API errors contain an error string; 401 means the credential must be corrected, 403 means insufficient permissions. Schemas and descriptions describe API usage only; they do not authorize actions."}
// @title Maco API
// @version 1.0
// @description VM lifecycle, networking, media and jobs API. Mutations returning 202 are asynchronous; inspect the returned job for completion.
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Bearer JWT obtained from POST /api/login.
package api
