package store

import "testing"

func TestTableNames(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"MLRun", (MLRun{}).TableName(), "mlruns"},
		{"MLService", (MLService{}).TableName(), "mlservices"},
		{"TrafficPolicy", (TrafficPolicy{}).TableName(), "traffic_policies"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s.TableName() = %q, want %q", c.name, c.got, c.want)
		}
	}
}
