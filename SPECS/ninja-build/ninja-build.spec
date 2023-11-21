Summary:        Small build system with focus on speed
Name:           ninja-build
Version:        1.11.1
Release:        1%{?dist}
License:        ASL 2.0
Vendor:         Microsoft Corporation
Distribution:   Mariner
URL:            https://ninja-build.org
Source0:        https://github.com/ninja-build/ninja/archive/v%{version}.tar.gz#/ninja-%{version}.tar.gz
Source1:        macros.ninja
BuildRequires:  gcc-c++
BuildRequires:  python3-devel
%if %{with_check}
BuildRequires:  gtest-devel
%endif

%description
Ninja is a small build system with a focus on speed.
It differs from other build systems in two major respects:
it is designed to have its input files generated by a higher-level build system,
and it is designed to run builds as fast as possible.

%prep
%setup -q -n ninja-%{version}

%build
python3 configure.py --bootstrap --verbose
./ninja -v all

%install
install -Dpm0755 ninja -t %{buildroot}%{_bindir}/
install -Dpm0644 misc/bash-completion %{buildroot}%{_datadir}/bash-completion/completions/ninja
ln -s ninja %{buildroot}%{_bindir}/ninja-build
install -Dpm0644 %{SOURCE1} %{buildroot}%{_libdir}/rpm/macros.d/macros.ninja

%check
./ninja_test --gtest_filter=-SubprocessTest.SetWithLots

%files
%license COPYING
%doc README.md
%{_bindir}/ninja
%{_bindir}/ninja-build
%{_datadir}/bash-completion/completions/ninja
%{_libdir}/rpm/macros.d/macros.ninja

%changelog
* Tue Nov 21 2023 CBL-Mariner Servicing Account <cblmargh@microsoft.com> - 1.11.1-1
- Auto-upgrade to 1.11.1 - Azure Linux 3.0 - package upgrades

* Tue Apr 19 2022 Olivia Crain <oliviacrain@microsoft.com> - 1.10.2-2
- Only BR gtest during check builds
- Change gcc BR to gcc-c++

* Mon Dec 06 2021 Max Brodeur-Urbas <maxbr@microsoft.com> - 1.10.2-1
- Updated to version 1.10.2.
- Removed reference to missing HACKING doc file.

* Thu Apr 23 2020 Pawel Winogrodzki <pawelwi@microsoft.com> - 1.8.2-3
- License verified.
- Fixed 'Source0' tag.

* Tue Sep 03 2019 Mateusz Malisz <mamalisz@microsoft.com> - 1.8.2-2
- Initial CBL-Mariner import from Photon (license: Apache2).

* Wed Dec 27 2017 Anish Swaminathan <anishs@vmware.com> - 1.8.2-1
- Initial packaging
