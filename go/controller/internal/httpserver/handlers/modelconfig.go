package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/kagent-dev/kagent/go/controller/api/v1alpha1"
	"github.com/kagent-dev/kagent/go/controller/internal/httpserver/errors"
	common "github.com/kagent-dev/kagent/go/controller/internal/utils"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// ModelConfigResponse defines the structure for the model config API response.
type ModelConfigResponse struct {
	Name            string                 `json:"name"`
	Namespace       string                 `json:"namespace"`
	ProviderName    string                 `json:"providerName"`
	Model           string                 `json:"model"`
	APIKeySecretRef string                 `json:"apiKeySecretRef"`
	APIKeySecretKey string                 `json:"apiKeySecretKey"`
	ModelParams     map[string]interface{} `json:"modelParams"`
}

// ModelConfigHandler handles model configuration requests
type ModelConfigHandler struct {
	*Base
}

// NewModelConfigHandler creates a new ModelConfigHandler
func NewModelConfigHandler(base *Base) *ModelConfigHandler {
	return &ModelConfigHandler{Base: base}
}

// HandleListModelConfigs handles GET /api/modelconfigs requests
func (h *ModelConfigHandler) HandleListModelConfigs(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("modelconfig-handler").WithValues("operation", "list")

	modelConfigs := &v1alpha1.ModelConfigList{}
	if err := h.KubeClient.List(r.Context(), modelConfigs); err != nil {
		w.RespondWithError(errors.NewInternalServerError("Failed to list model configs from Kubernetes", err))
		return
	}

	configs := make([]ModelConfigResponse, 0)
	for _, config := range modelConfigs.Items {
		log.V(1).Info("Processing model config", "name", config.Name, "model", config.Spec.Model)
		modelParams := make(map[string]interface{})

		if config.Spec.OpenAI != nil {
			FlattenStructToMap(config.Spec.OpenAI, modelParams)
		}
		if config.Spec.Anthropic != nil {
			FlattenStructToMap(config.Spec.Anthropic, modelParams)
		}
		if config.Spec.AzureOpenAI != nil {
			FlattenStructToMap(config.Spec.AzureOpenAI, modelParams)
		}
		if config.Spec.Ollama != nil {
			FlattenStructToMap(config.Spec.Ollama, modelParams)
		}
		if config.Spec.GeminiVertexAI != nil {
			FlattenStructToMap(config.Spec.GeminiVertexAI, modelParams)
		}
		if config.Spec.AnthropicVertexAI != nil {
			FlattenStructToMap(config.Spec.AnthropicVertexAI, modelParams)
		}

		responseItem := ModelConfigResponse{
			Name:            config.Name,
			Namespace:       config.Namespace,
			ProviderName:    string(config.Spec.Provider),
			Model:           config.Spec.Model,
			APIKeySecretRef: config.Spec.APIKeySecretRef,
			APIKeySecretKey: config.Spec.APIKeySecretKey,
			ModelParams:     modelParams,
		}
		configs = append(configs, responseItem)
	}

	log.Info("Successfully listed model configs", "count", len(configs))
	RespondWithJSON(w, http.StatusOK, configs)
}

// HandleGetModelConfig handles GET /api/modelconfigs/{configName} requests
func (h *ModelConfigHandler) HandleGetModelConfig(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("modelconfig-handler").WithValues("operation", "get")

	configName, err := GetPathParam(r, "configName")
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("Failed to get config name from path", err))
		return
	}
	log = log.WithValues("configName", configName)

	log.V(1).Info("Getting model config from Kubernetes")
	modelConfig := &v1alpha1.ModelConfig{}
	if err := h.KubeClient.Get(r.Context(), types.NamespacedName{
		Name:      configName,
		Namespace: common.GetResourceNamespace(),
	}, modelConfig); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("Model config not found")
			w.RespondWithError(errors.NewNotFoundError("Model config not found", nil))
			return
		}
		log.Error(err, "Failed to get model config")
		w.RespondWithError(errors.NewInternalServerError("Failed to get model config", err))
		return
	}

	log.V(1).Info("Constructing response object")
	modelParams := make(map[string]interface{})
	if modelConfig.Spec.OpenAI != nil {
		FlattenStructToMap(modelConfig.Spec.OpenAI, modelParams)
	}
	if modelConfig.Spec.Anthropic != nil {
		FlattenStructToMap(modelConfig.Spec.Anthropic, modelParams)
	}
	if modelConfig.Spec.AzureOpenAI != nil {
		FlattenStructToMap(modelConfig.Spec.AzureOpenAI, modelParams)
	}
	if modelConfig.Spec.Ollama != nil {
		FlattenStructToMap(modelConfig.Spec.Ollama, modelParams)
	}
	if modelConfig.Spec.GeminiVertexAI != nil {
		FlattenStructToMap(modelConfig.Spec.GeminiVertexAI, modelParams)
	}
	if modelConfig.Spec.AnthropicVertexAI != nil {
		FlattenStructToMap(modelConfig.Spec.AnthropicVertexAI, modelParams)
	}

	responseItem := ModelConfigResponse{
		Name:            modelConfig.Name,
		Namespace:       modelConfig.Namespace,
		ProviderName:    string(modelConfig.Spec.Provider),
		Model:           modelConfig.Spec.Model,
		APIKeySecretRef: modelConfig.Spec.APIKeySecretRef,
		APIKeySecretKey: modelConfig.Spec.APIKeySecretKey,
		ModelParams:     modelParams,
	}

	log.Info("Successfully retrieved and formatted model config")
	RespondWithJSON(w, http.StatusOK, responseItem)
}

// Helper function to get all JSON keys from a struct type
func getStructJSONKeys(structType reflect.Type) []string {
	keys := []string{}
	if structType.Kind() != reflect.Struct {
		return keys
	}
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" && jsonTag != "-" {
			tagParts := strings.Split(jsonTag, ",")
			keys = append(keys, tagParts[0])
		}
	}
	return keys
}

type CreateModelConfigRequest struct {
	Name                  string                            `json:"name"`
	Provider              Provider                          `json:"provider"`
	Model                 string                            `json:"model"`
	APIKey                string                            `json:"apiKey"`
	OpenAIParams          *v1alpha1.OpenAIConfig            `json:"openAI,omitempty"`
	AnthropicParams       *v1alpha1.AnthropicConfig         `json:"anthropic,omitempty"`
	AzureParams           *v1alpha1.AzureOpenAIConfig       `json:"azureOpenAI,omitempty"`
	OllamaParams          *v1alpha1.OllamaConfig            `json:"ollama,omitempty"`
	GeminiParams          *v1alpha1.GeminiVertexAIConfig    `json:"geminiVertexAI,omitempty"`
	AnthropicVertexParams *v1alpha1.AnthropicVertexAIConfig `json:"anthropicVertexAI,omitempty"`
}

type Provider struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func (h *ModelConfigHandler) HandleCreateModelConfig(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("modelconfig-handler").WithValues("operation", "create")

	var req CreateModelConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error(err, "Failed to decode request body")
		w.RespondWithError(errors.NewBadRequestError("Invalid request body", err))
		return
	}
	log = log.WithValues("configName", req.Name, "provider", req.Provider.Type, "model", req.Model)
	log.Info("Received request to create model config")

	log.V(1).Info("Checking if model config already exists")
	existingConfig := &v1alpha1.ModelConfig{}
	err := h.KubeClient.Get(r.Context(), types.NamespacedName{
		Name:      req.Name,
		Namespace: common.GetResourceNamespace(),
	}, existingConfig)
	if err == nil {
		log.Info("Model config already exists")
		w.RespondWithError(errors.NewConflictError("Model config already exists", nil))
		return
	} else if !k8serrors.IsNotFound(err) {
		log.Error(err, "Failed to check if model config exists")
		w.RespondWithError(errors.NewInternalServerError("Failed to check if model config exists", err))
		return
	}

	// --- Secret Creation ---
	providerTypeEnum := v1alpha1.ModelProvider(req.Provider.Type)
	modelConfigSpec := v1alpha1.ModelConfigSpec{
		Model:    req.Model,
		Provider: providerTypeEnum,
	}
	secret := &corev1.Secret{}

	// If the provider is Ollama, we don't need to create a secret.
	if providerTypeEnum == v1alpha1.Ollama || req.APIKey == "" {
		log.V(1).Info("Ollama provider or empty API key, skipping secret creation")
	} else {
		apiKey := req.APIKey
		secretName := req.Name
		secretKey := fmt.Sprintf("%s_API_KEY", strings.ToUpper(req.Provider.Type))
		log.V(1).Info("Creating API key secret", "secretName", secretName, "secretKey", secretKey)
		secret, err = CreateSecret(h.KubeClient, secretName, common.GetResourceNamespace(), map[string]string{secretKey: apiKey})
		if err != nil {
			log.Error(err, "Failed to create API key secret")
			w.RespondWithError(errors.NewInternalServerError("Failed to create API key secret", err))
			return
		}
		log.V(1).Info("Successfully created API key secret")
		modelConfigSpec.APIKeySecretRef = secretName
		modelConfigSpec.APIKeySecretKey = secretKey
	}

	modelConfig := &v1alpha1.ModelConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: common.GetResourceNamespace(),
		},
		Spec: modelConfigSpec,
	}

	var providerConfigErr error
	switch providerTypeEnum {
	case v1alpha1.OpenAI:
		if req.OpenAIParams != nil {
			modelConfig.Spec.OpenAI = req.OpenAIParams
			log.V(1).Info("Assigned OpenAI params to spec")
		} else {
			log.V(1).Info("No OpenAI params provided in create.")
		}
	case v1alpha1.Anthropic:
		if req.AnthropicParams != nil {
			modelConfig.Spec.Anthropic = req.AnthropicParams
			log.V(1).Info("Assigned Anthropic params to spec")
		} else {
			log.V(1).Info("No Anthropic params provided in create.")
		}
	case v1alpha1.AzureOpenAI:
		if req.AzureParams == nil {
			providerConfigErr = fmt.Errorf("azureOpenAI parameters are required for AzureOpenAI provider")
		} else {
			// Basic validation for required Azure fields (can be enhanced)
			if req.AzureParams.Endpoint == "" || req.AzureParams.APIVersion == "" {
				providerConfigErr = fmt.Errorf("missing required AzureOpenAI parameters: azureEndpoint, apiVersion")
			} else {
				modelConfig.Spec.AzureOpenAI = req.AzureParams
				log.V(1).Info("Assigned AzureOpenAI params to spec")
			}
		}
	case v1alpha1.Ollama:
		if req.OllamaParams != nil {
			modelConfig.Spec.Ollama = req.OllamaParams
			log.V(1).Info("Assigned Ollama params to spec")
		} else {
			log.V(1).Info("No Ollama params provided in create.")
		}
	case v1alpha1.GeminiVertexAI:
		if req.GeminiParams == nil {
			providerConfigErr = fmt.Errorf("geminiVertexAI parameters are required for GeminiVertexAI provider")
		} else {
			// Basic validation for required Gemini Vertex AI fields
			if req.GeminiParams.ProjectID == "" || req.GeminiParams.Location == "" {
				providerConfigErr = fmt.Errorf("missing required GeminiVertexAI parameters: projectID, location")
			} else {
				modelConfig.Spec.GeminiVertexAI = req.GeminiParams
				log.V(1).Info("Assigned GeminiVertexAI params to spec")
			}
		}
	case v1alpha1.AnthropicVertexAI:
		if req.AnthropicVertexParams == nil {
			providerConfigErr = fmt.Errorf("anthropicVertexAI parameters are required for AnthropicVertexAI provider")
		} else {
			// Basic validation for required Anthropic Vertex AI fields
			if req.AnthropicVertexParams.ProjectID == "" || req.AnthropicVertexParams.Location == "" {
				providerConfigErr = fmt.Errorf("missing required AnthropicVertexAI parameters: projectID, location")
			} else {
				modelConfig.Spec.AnthropicVertexAI = req.AnthropicVertexParams
				log.V(1).Info("Assigned AnthropicVertexAI params to spec")
			}
		}
	default:
		providerConfigErr = fmt.Errorf("unsupported provider type: %s", req.Provider.Type)
	}

	if providerConfigErr != nil {
		log.Error(providerConfigErr, "Failed to assign provider config")
		// Clean up the created secret if config assignment fails
		log.V(1).Info("Attempting to clean up secret due to config assignment failure")
		if providerTypeEnum != v1alpha1.Ollama {
			if cleanupErr := h.KubeClient.Delete(r.Context(), secret); cleanupErr != nil {
				log.Error(cleanupErr, "Failed to cleanup secret after config assignment failure")
			}
		}
		w.RespondWithError(errors.NewBadRequestError(providerConfigErr.Error(), providerConfigErr))
		return
	}

	if err := h.KubeClient.Create(r.Context(), modelConfig); err != nil {
		log.Error(err, "Failed to create ModelConfig resource")
		// If we fail to create the ModelConfig, we should clean up the secret
		log.V(1).Info("Attempting to clean up secret after ModelConfig creation failure")
		if providerTypeEnum != v1alpha1.Ollama {
			if cleanupErr := h.KubeClient.Delete(r.Context(), secret); cleanupErr != nil {
				log.Error(cleanupErr, "Failed to cleanup secret after ModelConfig creation failure")
			}
		}
		w.RespondWithError(errors.NewInternalServerError("Failed to create model config", err))
		return
	}

	log.Info("Successfully created model config", "name", req.Name)
	RespondWithJSON(w, http.StatusCreated, modelConfig)
}

// UpdateModelConfigRequest defines the structure for updating a model config.
// It's similar to Create, but APIKey is optional.
type UpdateModelConfigRequest struct {
	Provider              Provider                          `json:"provider"`
	Model                 string                            `json:"model"`
	APIKey                *string                           `json:"apiKey,omitempty"`
	OpenAIParams          *v1alpha1.OpenAIConfig            `json:"openAI,omitempty"`
	AnthropicParams       *v1alpha1.AnthropicConfig         `json:"anthropic,omitempty"`
	AzureParams           *v1alpha1.AzureOpenAIConfig       `json:"azureOpenAI,omitempty"`
	OllamaParams          *v1alpha1.OllamaConfig            `json:"ollama,omitempty"`
	GeminiParams          *v1alpha1.GeminiVertexAIConfig    `json:"geminiVertexAI,omitempty"`
	AnthropicVertexParams *v1alpha1.AnthropicVertexAIConfig `json:"anthropicVertexAI,omitempty"`
}

func (h *ModelConfigHandler) HandleUpdateModelConfig(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("modelconfig-handler").WithValues("operation", "update")

	configName, err := GetPathParam(r, "configName")
	if err != nil {
		log.Error(err, "Failed to get config name from path")
		w.RespondWithError(errors.NewBadRequestError("Failed to get config name from path", err))
		return
	}
	log = log.WithValues("configName", configName)

	var req UpdateModelConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error(err, "Failed to decode request body")
		w.RespondWithError(errors.NewBadRequestError("Invalid request body", err))
		return
	}
	log = log.WithValues("provider", req.Provider.Type, "model", req.Model)
	log.Info("Received request to update model config")

	log.V(1).Info("Getting existing model config")
	modelConfig := &v1alpha1.ModelConfig{}
	if err := h.KubeClient.Get(r.Context(), types.NamespacedName{
		Name:      configName,
		Namespace: common.GetResourceNamespace(),
	}, modelConfig); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("Model config not found")
			w.RespondWithError(errors.NewNotFoundError("Model config not found", nil))
			return
		}
		log.Error(err, "Failed to get model config")
		w.RespondWithError(errors.NewInternalServerError("Failed to get model config", err))
		return
	}

	modelConfig.Spec = v1alpha1.ModelConfigSpec{
		Model:       req.Model,
		Provider:    v1alpha1.ModelProvider(req.Provider.Type),
		OpenAI:      nil,
		Anthropic:   nil,
		AzureOpenAI: nil,
		Ollama:      nil,
	}

	// --- Update Secret if API Key is provided (and not Ollama or using AI API Gateway) ---
	shouldUpdateSecret := req.APIKey != nil && *req.APIKey != "" && modelConfig.Spec.Provider != v1alpha1.Ollama
	if shouldUpdateSecret {
		secretName := configName
		secretKey := fmt.Sprintf("%s_API_KEY", strings.ToUpper(req.Provider.Type))
		log.V(1).Info("Updating API key secret", "secretName", secretName, "secretKey", secretKey)
		existingSecret := &corev1.Secret{}
		err = h.KubeClient.Get(r.Context(), types.NamespacedName{Name: secretName, Namespace: common.GetResourceNamespace()}, existingSecret)
		if err != nil && !k8serrors.IsNotFound(err) {
			log.Error(err, "Failed to get existing secret for update")
			w.RespondWithError(errors.NewInternalServerError("Failed to get API key secret", err))
			return
		}

		if k8serrors.IsNotFound(err) {
			// Secret doesn't exist, create it (edge case, should normally exist)
			log.Info("Secret not found for update, creating new one", "secretName", secretName)
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: common.GetResourceNamespace()},
				StringData: map[string]string{secretKey: *req.APIKey},
			}
			if err := h.KubeClient.Create(r.Context(), secret); err != nil {
				log.Error(err, "Failed to create new API key secret during update")
				w.RespondWithError(errors.NewInternalServerError("Failed to create API key secret", err))
				return
			}
		} else {
			// Secret exists, update it
			if existingSecret.StringData == nil {
				existingSecret.StringData = make(map[string]string)
			}
			existingSecret.StringData[secretKey] = *req.APIKey
			if err := h.KubeClient.Update(r.Context(), existingSecret); err != nil {
				log.Error(err, "Failed to update API key secret")
				w.RespondWithError(errors.NewInternalServerError("Failed to update API key secret", err))
				return
			}
		}
		log.V(1).Info("Successfully updated API key secret")
		modelConfig.Spec.APIKeySecretRef = secretName
		modelConfig.Spec.APIKeySecretKey = secretKey
	}

	var providerConfigErr error
	switch modelConfig.Spec.Provider {
	case v1alpha1.OpenAI:
		if req.OpenAIParams != nil {
			modelConfig.Spec.OpenAI = req.OpenAIParams
			log.V(1).Info("Assigned updated OpenAI params to spec")
		} else {
			log.V(1).Info("No OpenAI params provided in update.")
		}
	case v1alpha1.Anthropic:
		if req.AnthropicParams != nil {
			modelConfig.Spec.Anthropic = req.AnthropicParams
			log.V(1).Info("Assigned updated Anthropic params to spec")
		} else {
			log.V(1).Info("No Anthropic params provided in update.")
		}
	case v1alpha1.AzureOpenAI:
		if req.AzureParams == nil {
			// Allow clearing Azure params if provider changes AWAY from Azure,
			// but require params if provider IS Azure.
			providerConfigErr = fmt.Errorf("azureOpenAI parameters are required when provider is AzureOpenAI")
		} else {
			// Basic validation for required Azure fields
			if req.AzureParams.Endpoint == "" || req.AzureParams.APIVersion == "" {
				providerConfigErr = fmt.Errorf("missing required AzureOpenAI parameters: azureEndpoint, apiVersion")
			} else {
				modelConfig.Spec.AzureOpenAI = req.AzureParams
				log.V(1).Info("Assigned updated AzureOpenAI params to spec")
			}
		}
	case v1alpha1.Ollama:
		if req.OllamaParams != nil {
			modelConfig.Spec.Ollama = req.OllamaParams
			log.V(1).Info("Assigned updated Ollama params to spec")
		} else {
			log.V(1).Info("No Ollama params provided in update.")
		}
	case v1alpha1.GeminiVertexAI:
		if req.GeminiParams == nil {
			providerConfigErr = fmt.Errorf("geminiVertexAI parameters are required for GeminiVertexAI provider")
		} else {
			// Basic validation for required Gemini Vertex AI fields
			if req.GeminiParams.ProjectID == "" || req.GeminiParams.Location == "" {
				providerConfigErr = fmt.Errorf("missing required GeminiVertexAI parameters: projectID, location")
			} else {
				modelConfig.Spec.GeminiVertexAI = req.GeminiParams
				log.V(1).Info("Assigned updated GeminiVertexAI params to spec")
			}
		}
	case v1alpha1.AnthropicVertexAI:
		if req.AnthropicVertexParams == nil {
			providerConfigErr = fmt.Errorf("anthropicVertexAI parameters are required for AnthropicVertexAI provider")
		} else {
			// Basic validation for required Anthropic Vertex AI fields
			if req.AnthropicVertexParams.ProjectID == "" || req.AnthropicVertexParams.Location == "" {
				providerConfigErr = fmt.Errorf("missing required AnthropicVertexAI parameters: projectID, location")
			} else {
				modelConfig.Spec.AnthropicVertexAI = req.AnthropicVertexParams
				log.V(1).Info("Assigned updated AnthropicVertexAI params to spec")
			}
		}
	default:
		providerConfigErr = fmt.Errorf("unsupported provider type specified: %s", req.Provider.Type)
	}

	if providerConfigErr != nil {
		log.Error(providerConfigErr, "Failed to assign provider config during update")
		w.RespondWithError(errors.NewBadRequestError(providerConfigErr.Error(), providerConfigErr))
		return
	}

	if err := h.KubeClient.Update(r.Context(), modelConfig); err != nil {
		log.Error(err, "Failed to update ModelConfig resource")
		w.RespondWithError(errors.NewInternalServerError("Failed to update model config", err))
		return
	}

	log.Info("Successfully updated model config", "name", configName)
	updatedParams := make(map[string]interface{})
	if modelConfig.Spec.OpenAI != nil {
		FlattenStructToMap(modelConfig.Spec.OpenAI, updatedParams)
	} else if modelConfig.Spec.Anthropic != nil {
		FlattenStructToMap(modelConfig.Spec.Anthropic, updatedParams)
	} else if modelConfig.Spec.AzureOpenAI != nil {
		FlattenStructToMap(modelConfig.Spec.AzureOpenAI, updatedParams)
	} else if modelConfig.Spec.Ollama != nil {
		FlattenStructToMap(modelConfig.Spec.Ollama, updatedParams)
	} else if modelConfig.Spec.GeminiVertexAI != nil {
		FlattenStructToMap(modelConfig.Spec.GeminiVertexAI, updatedParams)
	} else if modelConfig.Spec.AnthropicVertexAI != nil {
		FlattenStructToMap(modelConfig.Spec.AnthropicVertexAI, updatedParams)
	}

	responseItem := ModelConfigResponse{
		Name:            modelConfig.Name,
		Namespace:       modelConfig.Namespace,
		ProviderName:    string(modelConfig.Spec.Provider),
		Model:           modelConfig.Spec.Model,
		APIKeySecretRef: modelConfig.Spec.APIKeySecretRef,
		APIKeySecretKey: modelConfig.Spec.APIKeySecretKey,
		ModelParams:     updatedParams,
	}
	RespondWithJSON(w, http.StatusOK, responseItem)
}

func (h *ModelConfigHandler) HandleDeleteModelConfig(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("modelconfig-handler").WithValues("operation", "delete")

	configName, err := GetPathParam(r, "configName")
	if err != nil {
		log.Error(err, "Failed to get config name from path")
		w.RespondWithError(errors.NewBadRequestError("Failed to get config name from path", err))
		return
	}
	log = log.WithValues("configName", configName)

	log.Info("Received request to delete model config")

	log.V(1).Info("Checking if model config exists")
	existingConfig := &v1alpha1.ModelConfig{}
	err = h.KubeClient.Get(r.Context(), types.NamespacedName{
		Name:      configName,
		Namespace: common.GetResourceNamespace(),
	}, existingConfig)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("Model config not found")
			w.RespondWithError(errors.NewNotFoundError("Model config not found", nil))
			return
		}
		log.Error(err, "Failed to get model config")
		w.RespondWithError(errors.NewInternalServerError("Failed to get model config", err))
		return
	}

	log.V(1).Info("Deleting ModelConfig resource")
	if err := h.KubeClient.Delete(r.Context(), existingConfig); err != nil {
		log.Error(err, "Failed to delete ModelConfig resource")
		w.RespondWithError(errors.NewInternalServerError("Failed to delete model config", err))
		return
	}

	log.Info("Successfully deleted model config", "name", configName)
	RespondWithJSON(w, http.StatusOK, nil)
}
