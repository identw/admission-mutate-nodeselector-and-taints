package mutate

import (
	"encoding/json"
	"fmt"
	"log"

	v1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PatchOperation struct {
    From  string        `json:"from,omitempty"`
    Op    string        `json:"op"`
    Path  string        `json:"path"`
    Value interface{}   `json:"value,omitempty"`
}

type Mutate struct {
	NodeSelector map[string]string   `json:"nodeselector"`
	Tolerations []corev1.Toleration  `json:"tolerations"`
	RemoveNodeAffinity bool          `json:"remove_node_affinity,omitempty"`
}

// Mutate mutates
func (m Mutate) Mutate(body []byte, verbose bool) ([]byte, error) {
	if verbose {
		log.Printf("recv: %s\n", string(body)) // untested section
	}

	// unmarshal request into AdmissionReview struct
	admReview := v1beta1.AdmissionReview{}
	if err := json.Unmarshal(body, &admReview); err != nil {
		return nil, fmt.Errorf("unmarshaling request failed with %s", err)
	}

	var err error
	var pod *corev1.Pod

	responseBody := []byte{}
	ar := admReview.Request
	resp := v1beta1.AdmissionResponse{}

	if ar != nil {

		// get the Pod object and unmarshal it into its struct, if we cannot, we might as well stop here
		if err := json.Unmarshal(ar.Object.Raw, &pod); err != nil {
			return nil, fmt.Errorf("unable unmarshal pod json object %v", err)
		}
		
		// set response options
		resp.Allowed = true
		resp.UID = ar.UID
		pT := v1beta1.PatchTypeJSONPatch
		resp.PatchType = &pT // it's annoying that this needs to be a pointer as you cannot give a pointer to a constant?

		// add some audit annotations, helpful to know why a object was modified, maybe (?)
		resp.AuditAnnotations = map[string]string{
			"mutateme": "add nodeSelector and taints",
		}

		// the actual mutation is done by a string in JSONPatch style, i.e. we don't _actually_ modify the object, but
		// tell K8S how it should modifiy it
		p := []PatchOperation{}
		patch := PatchOperation{}
		
		// remove affinity
		if m.RemoveNodeAffinity {
			if pod.Spec.Affinity != nil && pod.Spec.Affinity.NodeAffinity != nil {
				patch = PatchOperation{
					Op: "remove",
					Path: "/spec/affinity/nodeAffinity",
				}
				p = append(p, patch)
			}
		}

		// Add nodeSelector
		patch = PatchOperation{
			Op: "add",
			Path: "/spec/nodeSelector",
			Value: m.NodeSelector,
		}
		p = append(p, patch)

		// Add tolerations
		patch = PatchOperation{
			Op: "add",
			Path: "/spec/tolerations",
			Value: m.Tolerations,
		}
		p = append(p, patch)

		// parse the []map into JSON
		resp.Patch, err = json.Marshal(p)

		// Success, of course ;)
		resp.Result = &metav1.Status{
			Status: "Success",
		}

		admReview.Response = &resp
		// back into JSON so we can return the finished AdmissionReview w/ Response directly
		// w/o needing to convert things in the http handler
		responseBody, err = json.Marshal(admReview)
		if err != nil {
			return nil, err // untested section
		}
	}

	if verbose {
		log.Printf("resp: %s\n", string(responseBody)) // untested section
	}

	return responseBody, nil
}