package handler

import (
	"log/slog"
	"net/http"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/paymentmethod"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/toast"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

type MethodHandler struct {
	spaceService  *service.SpaceService
	methodService *service.PaymentMethodService
}

func NewMethodHandler(ss *service.SpaceService, pms *service.PaymentMethodService) *MethodHandler {
	return &MethodHandler{
		spaceService:  ss,
		methodService: pms,
	}
}

func (h *MethodHandler) getMethodForSpace(w http.ResponseWriter, spaceID, methodID string) *model.PaymentMethod {
	method, err := h.methodService.GetMethod(methodID)
	if err != nil {
		http.Error(w, "Payment method not found", http.StatusNotFound)
		return nil
	}
	if method.SpaceID != spaceID {
		http.Error(w, "Not Found", http.StatusNotFound)
		return nil
	}
	return method
}

func (h *MethodHandler) PaymentMethodsPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		http.Error(w, "Space not found", http.StatusNotFound)
		return
	}

	methods, err := h.methodService.GetMethodsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get payment methods for space", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.SpacePaymentMethodsPage(space, methods))
}

func (h *MethodHandler) CreatePaymentMethod(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	user := ctxkeys.User(r.Context())

	if err := r.ParseForm(); err != nil {
		ui.RenderError(w, r, "Bad Request", http.StatusUnprocessableEntity)
		return
	}

	name := r.FormValue("name")
	methodType := model.PaymentMethodType(r.FormValue("type"))
	lastFour := r.FormValue("last_four")

	method, err := h.methodService.CreateMethod(service.CreatePaymentMethodDTO{
		SpaceID:   spaceID,
		Name:      name,
		Type:      methodType,
		LastFour:  lastFour,
		CreatedBy: user.ID,
	})
	if err != nil {
		slog.Error("failed to create payment method", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	ui.Render(w, r, paymentmethod.MethodItem(spaceID, method))
}

func (h *MethodHandler) UpdatePaymentMethod(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	methodID := r.PathValue("methodID")

	if h.getMethodForSpace(w, spaceID, methodID) == nil {
		return
	}

	if err := r.ParseForm(); err != nil {
		ui.RenderError(w, r, "Bad Request", http.StatusUnprocessableEntity)
		return
	}

	name := r.FormValue("name")
	methodType := model.PaymentMethodType(r.FormValue("type"))
	lastFour := r.FormValue("last_four")

	updatedMethod, err := h.methodService.UpdateMethod(service.UpdatePaymentMethodDTO{
		ID:       methodID,
		Name:     name,
		Type:     methodType,
		LastFour: lastFour,
	})
	if err != nil {
		slog.Error("failed to update payment method", "error", err, "method_id", methodID)
		ui.RenderError(w, r, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	ui.Render(w, r, paymentmethod.MethodItem(spaceID, updatedMethod))
}

func (h *MethodHandler) DeletePaymentMethod(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	methodID := r.PathValue("methodID")

	if h.getMethodForSpace(w, spaceID, methodID) == nil {
		return
	}

	err := h.methodService.DeleteMethod(methodID)
	if err != nil {
		slog.Error("failed to delete payment method", "error", err, "method_id", methodID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	ui.RenderToast(w, r, toast.Toast(toast.Props{
		Title:       "Payment method deleted",
		Variant:     toast.VariantSuccess,
		Icon:        true,
		Dismissible: true,
		Duration:    5000,
	}))
}
