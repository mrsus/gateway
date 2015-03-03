package admin

import (
	"errors"
	"fmt"
	"gateway/config"
	aphttp "gateway/http"
	"gateway/model"
	apsql "gateway/sql"
	"net/http"

	"github.com/gorilla/handlers"
)

// RouteAPIExport routes the endpoint for API export
func RouteAPIExport(controller *APIsController, path string,
	router aphttp.Router, db *apsql.DB, conf config.ProxyAdmin) {

	routes := map[string]http.Handler{
		"GET": read(db, controller.Export),
	}
	if conf.CORSEnabled {
		routes["OPTIONS"] = aphttp.CORSOptionsHandler([]string{"GET", "OPTIONS"})
	}

	router.Handle(path, handlers.MethodHandler(routes))
}

// RouteAPIImport routes the endpoint for API import
func RouteAPIImport(controller *APIsController, path string,
	router aphttp.Router, db *apsql.DB, conf config.ProxyAdmin) {

	routes := map[string]http.Handler{
		"POST": write(db, controller.Import),
	}
	if conf.CORSEnabled {
		routes["OPTIONS"] = aphttp.CORSOptionsHandler([]string{"POST", "OPTIONS"})
	}

	router.Handle(path, handlers.MethodHandler(routes))
}

// Export exports a whole API
func (c *APIsController) Export(w http.ResponseWriter, r *http.Request,
	db *apsql.DB) aphttp.Error {

	id := instanceID(r)

	api, err := model.FindAPIForAccountIDForExport(db, id, c.accountID(r))

	if err != nil {
		return c.notFound()
	}

	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s.json"`, api.Name))

	return c.serializeInstance(api, w)
}

// Import imports a full API
func (c *APIsController) Import(w http.ResponseWriter, r *http.Request,
	tx *apsql.Tx) aphttp.Error {

	export, _, err := r.FormFile("export")
	if err != nil {
		return aphttp.NewError(errors.New("Could not get file from 'export' field."),
			http.StatusBadRequest)
	}

	api, httpErr := c.deserializeInstance(export)
	if httpErr != nil {
		return httpErr
	}

	api.Name = r.FormValue("name")
	api.AccountID = c.accountID(r)

	validationErrors := api.Validate()
	if !validationErrors.Empty() {
		return SerializableValidationErrors{validationErrors}
	}

	if err := api.Import(tx); err != nil {
		validationErrors = api.ValidateFromDatabaseError(err)
		if !validationErrors.Empty() {
			return SerializableValidationErrors{validationErrors}
		}
		return aphttp.NewServerError(err)
	}

	return c.serializeInstance(api, w)
}
