Name:           tidy
Version:        1.1.0
Release:        1%{?dist}
Summary:        Smart file organizer for your terminal
License:        MIT
URL:            https://github.com/YousefMohiey/tidy
Source0:        https://github.com/YousefMohiey/tidy/releases/download/v%{version}/tidy-Linux-amd64

%description
tidy automatically organizes files into categorized directories based on
extension, MIME type, and filename patterns. Features content-aware sorting,
interactive TUI dashboard, duplicate detection, real-time watch mode, and
full undo support. Supports 300+ file types across 11 categories.

%prep
# Binary distribution - nothing to prep

%build
# Pre-built binary

%install
mkdir -p %{buildroot}%{_bindir}
install -m 755 %{SOURCE0} %{buildroot}%{_bindir}/tidy
mkdir -p %{buildroot}%{_datadir}/icons/hicolor/256x256/apps
install -m 644 %{_sourcedir}/tidy.png %{buildroot}%{_datadir}/icons/hicolor/256x256/apps/tidy.png
mkdir -p %{buildroot}%{_datadir}/applications
cat > %{buildroot}%{_datadir}/applications/tidy.desktop << 'EOF'
[Desktop Entry]
Name=Tidy
Comment=Smart file organizer
Exec=tidy
Icon=tidy
Terminal=true
Type=Application
Categories=Utility;FileManager;
Keywords=files;organize;sort;cleanup;
EOF

%files
%{_bindir}/tidy
%{_datadir}/icons/hicolor/256x256/apps/tidy.png
%{_datadir}/applications/tidy.desktop

%changelog
* Mon Jun 09 2026 YousefMohiey <yousefmohiey@gmail.com> - 1.1.0-1
- Initial RPM package
