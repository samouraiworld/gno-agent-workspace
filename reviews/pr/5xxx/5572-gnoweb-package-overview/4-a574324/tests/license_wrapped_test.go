/* Run: from a gno checkout:
gh pr checkout 5572 -R gnolang/gno && git checkout a574324
curl -fsSL -o gno.land/pkg/gnoweb/components/zz_license_wrapped_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5572-gnoweb-package-overview/4-a574324/tests/license_wrapped_test.go
go test -v -run TestDeriveLicense_WrappedRealWorldFiles ./gno.land/pkg/gnoweb/components/
rm gno.land/pkg/gnoweb/components/zz_license_wrapped_test.go
*/

// The AGPL, GPL and BSD signatures in licenseSignatures join their anchor phrases
// with ".*", and Go's "." never matches a newline, so a real LICENSE file that
// wraps between those words falls through to an empty Kind.
// At a574324 the four wrapped cases below report Kind "", while Apache-2.0 (which
// joins with "\s*") is detected; adding the "s" flag to those patterns turns all
// five green.

package components

import "testing"

func TestDeriveLicense_WrappedRealWorldFiles(t *testing.T) {
	t.Parallel()

	// Headers copied verbatim from real LICENSE files, line wrapping included.
	const (
		gpl3 = "                    GNU GENERAL PUBLIC LICENSE\n" +
			"                       Version 3, 29 June 2007\n\n" +
			" Copyright (C) 2007 Free Software Foundation, Inc. <https://fsf.org/>\n"

		agpl3 = "                    GNU AFFERO GENERAL PUBLIC LICENSE\n" +
			"                       Version 3, 19 November 2007\n"

		bsd3 = "Copyright (c) 2017 The Libc Authors. All rights reserved.\n\n" +
			"Redistribution and use in source and binary forms, with or without\n" +
			"modification, are permitted provided that the following conditions are\n" +
			"met:\n\n" +
			"3. Neither the name of the copyright holder\n"

		bsd2 = "Redistribution and use in source and binary forms, with or without\n" +
			"modification, are permitted provided that the following conditions are met:\n"

		// Detected today: "apache license\s*,?\s*version 2\.0" crosses the newline.
		apache2 = "                                 Apache License\n" +
			"                           Version 2.0, January 2004\n"
	)

	cases := []struct{ name, body, want string }{
		{"GPL-3.0", gpl3, "GPL-3.0"},
		{"AGPL-3.0", agpl3, "AGPL-3.0"},
		{"BSD-3-Clause", bsd3, "BSD-3-Clause"},
		{"BSD-2-Clause", bsd2, "BSD-2-Clause"},
		{"Apache-2.0 baseline", apache2, "Apache-2.0"},
	}

	for _, tc := range cases {
		got := deriveLicense([]string{"LICENSE"}, fileContentFn(map[string][]byte{"LICENSE": []byte(tc.body)}))
		if got.Kind != tc.want {
			t.Errorf("%s: a real line-wrapped LICENSE reports Kind %q, want %q", tc.name, got.Kind, tc.want)
		}
	}
}
