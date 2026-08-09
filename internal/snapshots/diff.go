package snapshots

import (
	"fmt"
	"reflect"
)

// Compare produces a readable Diff between two snapshots (left → right).
func Compare(left, right Snapshot) Diff {
	d := Diff{
		LeftID: left.ID, RightID: right.ID,
		LeftAt: left.CreatedAt, RightAt: right.CreatedAt,
	}
	d.Containers = diffContainers(left.Containers, right.Containers)
	d.Images = diffImages(left.Images, right.Images)
	d.Networks = diffNetworks(left.Networks, right.Networks)
	d.Volumes = diffVolumes(left.Volumes, right.Volumes)
	d.Stacks = diffStacks(left.Stacks, right.Stacks)
	return d
}

func diffContainers(a, b []ContainerView) []Change {
	am := map[string]ContainerView{}
	bm := map[string]ContainerView{}
	for _, x := range a {
		am[x.ID] = x
	}
	for _, x := range b {
		bm[x.ID] = x
	}
	var out []Change
	for id, x := range bm {
		if _, ok := am[id]; !ok {
			out = append(out, Change{Kind: ChangeAdded, ID: id, Name: x.Name})
		}
	}
	for id, x := range am {
		if _, ok := bm[id]; !ok {
			out = append(out, Change{Kind: ChangeRemoved, ID: id, Name: x.Name})
			continue
		}
		y := bm[id]
		fields := []FieldChange{}
		fields = appendField(fields, "image", x.Image, y.Image)
		fields = appendField(fields, "state", string(x.State), string(y.State))
		fields = appendField(fields, "health", string(x.Health), string(y.Health))
		fields = appendField(fields, "restartCount", x.RestartCount, y.RestartCount)
		fields = appendField(fields, "writableBytes", x.WritableBytes, y.WritableBytes)
		fields = appendField(fields, "memoryBytes", x.MemoryBytes, y.MemoryBytes)
		if len(fields) > 0 {
			out = append(out, Change{Kind: ChangeModified, ID: id, Name: y.Name, Fields: fields})
		}
	}
	return out
}

func diffImages(a, b []ImageView) []Change {
	am := map[string]ImageView{}
	bm := map[string]ImageView{}
	for _, x := range a {
		am[x.ID] = x
	}
	for _, x := range b {
		bm[x.ID] = x
	}
	var out []Change
	for id, x := range bm {
		if _, ok := am[id]; !ok {
			out = append(out, Change{Kind: ChangeAdded, ID: id, Name: imageName(x)})
		}
	}
	for id, x := range am {
		if _, ok := bm[id]; !ok {
			out = append(out, Change{Kind: ChangeRemoved, ID: id, Name: imageName(x)})
			continue
		}
		y := bm[id]
		fields := []FieldChange{}
		fields = appendField(fields, "sizeBytes", x.SizeBytes, y.SizeBytes)
		fields = appendField(fields, "dangling", x.Dangling, y.Dangling)
		fields = appendField(fields, "containerCount", x.ContainerCount, y.ContainerCount)
		if len(fields) > 0 {
			out = append(out, Change{Kind: ChangeModified, ID: id, Name: imageName(y), Fields: fields})
		}
	}
	return out
}

func diffNetworks(a, b []NetworkView) []Change {
	am := map[string]NetworkView{}
	bm := map[string]NetworkView{}
	for _, x := range a {
		am[x.ID] = x
	}
	for _, x := range b {
		bm[x.ID] = x
	}
	var out []Change
	for id, x := range bm {
		if _, ok := am[id]; !ok {
			out = append(out, Change{Kind: ChangeAdded, ID: id, Name: x.Name})
		}
	}
	for id, x := range am {
		if y, ok := bm[id]; !ok {
			out = append(out, Change{Kind: ChangeRemoved, ID: id, Name: x.Name})
		} else if !reflect.DeepEqual(x.Containers, y.Containers) || x.Driver != y.Driver {
			fields := []FieldChange{}
			fields = appendField(fields, "driver", x.Driver, y.Driver)
			fields = appendField(fields, "containers", x.Containers, y.Containers)
			out = append(out, Change{Kind: ChangeModified, ID: id, Name: y.Name, Fields: fields})
		}
	}
	return out
}

func diffVolumes(a, b []VolumeView) []Change {
	am := map[string]VolumeView{}
	bm := map[string]VolumeView{}
	for _, x := range a {
		am[x.Name] = x
	}
	for _, x := range b {
		bm[x.Name] = x
	}
	var out []Change
	for name := range bm {
		if _, ok := am[name]; !ok {
			out = append(out, Change{Kind: ChangeAdded, ID: name, Name: name})
		}
	}
	for name, x := range am {
		if y, ok := bm[name]; !ok {
			out = append(out, Change{Kind: ChangeRemoved, ID: name, Name: name})
		} else {
			fields := []FieldChange{}
			fields = appendField(fields, "usageBytes", x.UsageBytes, y.UsageBytes)
			fields = appendField(fields, "containers", x.Containers, y.Containers)
			if len(fields) > 0 {
				out = append(out, Change{Kind: ChangeModified, ID: name, Name: name, Fields: fields})
			}
		}
	}
	return out
}

func diffStacks(a, b []StackView) []Change {
	am := map[string]StackView{}
	bm := map[string]StackView{}
	for _, x := range a {
		am[x.Name] = x
	}
	for _, x := range b {
		bm[x.Name] = x
	}
	var out []Change
	for name := range bm {
		if _, ok := am[name]; !ok {
			out = append(out, Change{Kind: ChangeAdded, ID: name, Name: name})
		}
	}
	for name, x := range am {
		if y, ok := bm[name]; !ok {
			out = append(out, Change{Kind: ChangeRemoved, ID: name, Name: name})
		} else {
			fields := []FieldChange{}
			fields = appendField(fields, "containerCount", x.ContainerCount, y.ContainerCount)
			fields = appendField(fields, "runningCount", x.RunningCount, y.RunningCount)
			if len(fields) > 0 {
				out = append(out, Change{Kind: ChangeModified, ID: name, Name: name, Fields: fields})
			}
		}
	}
	return out
}

func appendField(fields []FieldChange, name string, from, to any) []FieldChange {
	if reflect.DeepEqual(from, to) {
		return fields
	}
	return append(fields, FieldChange{Field: name, From: from, To: to})
}

func imageName(img ImageView) string {
	if len(img.RepoTags) > 0 {
		return img.RepoTags[0]
	}
	return img.IDShort
}

// FormatChange is a compact human string for tests/UI helpers.
func FormatChange(c Change) string {
	switch c.Kind {
	case ChangeAdded:
		return fmt.Sprintf("+ %s", c.Name)
	case ChangeRemoved:
		return fmt.Sprintf("- %s", c.Name)
	default:
		return fmt.Sprintf("~ %s", c.Name)
	}
}
