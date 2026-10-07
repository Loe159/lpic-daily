Name: lpic-helper
Version: 1.0
Release: 1
Summary: LPIC Daily helper fixture
License: MIT
BuildArch: noarch

%description
Offline dependency fixture.

%install
mkdir -p %{buildroot}/usr/share/lpic-helper
printf 'helper-ok\n' > %{buildroot}/usr/share/lpic-helper/state.txt

%files
/usr/share/lpic-helper/state.txt
