package cron

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	jkt, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		expression string
		loc        *time.Location
		name       string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr string
	}{
		{
			name: "general expression", args: args{expression: "* * * * *", loc: jkt},
			want: `{ name:"general expression" schedule:"* * * * *", location:"Asia/Jakarta" }`, wantErr: "",
		},
		{
			name: "different timezone", args: args{expression: "* * * * *", loc: time.UTC},
			want: `{ name:"different timezone" schedule:"* * * * *", location:"UTC" }`, wantErr: "",
		},
		{
			name: "nil timezone", args: args{expression: "* * * * *", loc: nil},
			want: `{ name:"nil timezone" schedule:"* * * * *", location:"UTC" }`, wantErr: "",
		},
		{
			name: "normal value", args: args{expression: "59 23 31 12 6", loc: time.UTC},
			want: `{ name:"normal value" schedule:"59 23 31 12 6", location:"UTC" }`, wantErr: "",
		},
		{
			name: "invalid field", args: args{expression: "* * * *", loc: time.UTC}, want: ``,
			wantErr: "got 4 want 5 expressions",
		},
		{
			name: "wrong minute", args: args{expression: "60 23 31 12 6", loc: time.UTC}, want: ``,
			wantErr: `failed parsing 'minute' field "60": value out of range (0 - 59): 60`,
		},
		{
			name: "wrong hour", args: args{expression: "59 24 31 12 6", loc: time.UTC}, want: ``,
			wantErr: `failed parsing 'hour' field "24": value out of range (0 - 23): 24`,
		},
		{
			name: "wrong day of month 0", args: args{expression: "59 23 0 12 6", loc: time.UTC}, want: ``,
			wantErr: `failed parsing 'day of month' field "0": value out of range (1 - 31): 0`,
		},
		{
			name: "wrong day of month", args: args{expression: "59 23 32 12 6", loc: time.UTC}, want: ``,
			wantErr: `failed parsing 'day of month' field "32": value out of range (1 - 31): 32`,
		},
		{
			name: "wrong month 0", args: args{expression: "59 23 31 0 6", loc: time.UTC}, want: ``,
			wantErr: `failed parsing 'month' field "0": value out of range (1 - 12): 0`,
		},
		{
			name: "wrong month", args: args{expression: "59 23 31 13 6", loc: time.UTC}, want: ``,
			wantErr: `failed parsing 'month' field "13": value out of range (1 - 12): 13`,
		},
		{
			name: "wrong dow", args: args{expression: "59 23 31 12 7", loc: time.UTC}, want: ``,
			wantErr: `failed parsing 'day of week' field "7": value out of range (0 - 6): 7`,
		},
		{
			name: "with csv", args: args{expression: "1,3,5,7,9 23 31 12 6", loc: time.UTC},
			want: `{ name:"with csv" schedule:"1,3,5,7,9 23 31 12 6", location:"UTC" }`, wantErr: "",
		},
		{
			name: "with csv", args: args{expression: "1,3,60 23 31 12 6", loc: time.UTC}, want: ``,
			wantErr: `failed parsing 'minute' field "1,3,60": value out of range (0 - 59): 60`,
		},
		{
			name: "with step", args: args{expression: "*/2 23 31 12 6", loc: time.UTC},
			want: `{ name:"with step" schedule:"0,2,4,6,8,10,12,14,16,18,20,22,24,26,28,30,32,34,36,38,40,42,44,46,48,50,52,54,56,58 23 31 12 6", location:"UTC" }`, wantErr: "",
		},
		{
			name: "with step without range", args: args{expression: "30/2 23 31 12 7", loc: time.UTC}, want: ``,
			wantErr: `failed parsing 'minute' field "30/2": step given without range, expression "30/2"`,
		},
		{
			name: "with step and range", args: args{expression: "10-30/3 23 31 12 6", loc: time.UTC},
			want: `{ name:"with step and range" schedule:"10,13,16,19,22,25,28 23 31 12 6", location:"UTC" }`, wantErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			arg := tt.args
			arg.name = tt.name
			e, err := Parse(arg.expression, arg.loc, arg.name)
			if (err == nil) && tt.wantErr != "" || (err != nil) && err.Error() != tt.wantErr {
				t.Errorf("Parse() error = %q, wantErr %q", err, tt.wantErr)
				return
			}
			if tt.wantErr != "" {
				// test return the correct error
				return
			}

			got := e.String()
			if got != tt.want {
				t.Errorf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestMatch(t *testing.T) {
	jkt, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatal(err)
	}

	var zero time.Time
	type args struct {
		expression string
		loc        *time.Location
		name       string
	}
	tests := []struct {
		name         string
		args         args
		wantMatch    []time.Time
		wantNotMatch []time.Time
	}{
		{
			name: "match all", args: args{expression: "* * * * *", loc: time.UTC},
			wantMatch: []time.Time{
				time.Now(),
				zero,
				time.Date(2222, 12, 15, 15, 4, 5, 0, time.UTC),
			},
		},
		{
			name: "exact schedule", args: args{expression: "4 15 2 1 1", loc: time.UTC},
			wantMatch: []time.Time{
				time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
			},
			wantNotMatch: []time.Time{
				zero,
				time.Date(2006, 1, 2, 15, 3, 5, 0, time.UTC),  // min not match
				time.Date(2006, 1, 2, 15, 5, 5, 0, time.UTC),  // min not match
				time.Date(2006, 1, 2, 14, 4, 5, 0, time.UTC),  // hour not match
				time.Date(2006, 1, 2, 16, 4, 5, 0, time.UTC),  // hour not match
				time.Date(2006, 1, 1, 15, 4, 5, 0, time.UTC),  // day not match
				time.Date(2006, 1, 3, 15, 4, 5, 0, time.UTC),  // day not match
				time.Date(2006, 12, 2, 15, 4, 5, 0, time.UTC), // month not match
				time.Date(2006, 2, 2, 15, 4, 5, 0, time.UTC),  // month not match
				time.Date(2006, 1, 2, 15, 4, 5, 0, jkt),       // time zone not match
			},
		},
		{
			name: "multi entry", args: args{expression: "4,5 15,16 2,3 1,2 0,1,2,4", loc: time.UTC},
			wantMatch: []time.Time{
				time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
				time.Date(2006, 1, 2, 15, 5, 5, 0, time.UTC), // minute
				time.Date(2006, 1, 2, 16, 4, 5, 0, time.UTC), // hours
				time.Date(2006, 1, 3, 15, 4, 5, 0, time.UTC), // day of month
				time.Date(2006, 2, 2, 15, 4, 5, 0, time.UTC), // month
				time.Date(2006, 2, 2, 15, 4, 5, 0, time.UTC), // day of week (Thu)
				time.Date(2008, 2, 3, 15, 4, 5, 0, time.UTC), // sunday
				time.Date(2009, 2, 2, 15, 4, 5, 0, time.UTC), // monday
				time.Date(2010, 2, 2, 15, 4, 5, 0, time.UTC), // tuesday

			},
			wantNotMatch: []time.Time{
				zero,
				time.Date(2006, 1, 2, 15, 3, 5, 0, time.UTC),  // minute
				time.Date(2006, 1, 2, 15, 6, 5, 0, time.UTC),  // minute
				time.Date(2006, 1, 2, 14, 4, 5, 0, time.UTC),  // hours
				time.Date(2006, 1, 2, 17, 4, 5, 0, time.UTC),  // hours
				time.Date(2006, 1, 1, 15, 4, 5, 0, time.UTC),  // day of month
				time.Date(2006, 1, 4, 15, 4, 5, 0, time.UTC),  // day of month
				time.Date(2006, 12, 2, 15, 4, 5, 0, time.UTC), // month
				time.Date(2006, 3, 2, 15, 4, 5, 0, time.UTC),  // month
				time.Date(2015, 1, 2, 15, 4, 5, 0, time.UTC),  // friday
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			arg := tt.args
			arg.name = tt.name
			e, err := Parse(arg.expression, arg.loc, arg.name)
			if err != nil {
				t.Fatal(err)
			}

			for i, want := range tt.wantMatch {
				if !e.Match(want) {
					t.Errorf("[%d] want match %s with %s but it does not", i, e.String(), want)
				}
			}

			for i, want := range tt.wantNotMatch {
				if e.Match(want) {
					t.Errorf("[%d] want not match %s with %s but it does", i, e.String(), want)
				}
			}
		})
	}
}
