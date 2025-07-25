package inspektorgadget

import "testing"

func TestGadgets(t *testing.T) {
	for _, gadget := range gadgets {
		if gadget.Name == "" {
			t.Errorf("Gadget name is empty")
		}
		if gadget.Image == "" {
			t.Errorf("Gadget image is empty for %s", gadget.Name)
		}
		if gadget.Description == "" {
			t.Errorf("Gadget description is empty for %s", gadget.Name)
		}
	}
}
