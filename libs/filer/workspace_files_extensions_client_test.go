package filer

import (
	"context"
	"net/http"
	"testing"

	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocks client.DatabricksClient from the databricks-sdk-go package.
type mockApiClient struct {
	mock.Mock
}

func (m *mockApiClient) Do(ctx context.Context, method, path string,
	headers map[string]string, request any, response any,
	visitors ...func(*http.Request) error) error {
	args := m.Called(ctx, method, path, headers, request, response, visitors)

	// Set the http response from a value provided in the mock call.
	p := response.(*wsfsFileInfo)
	*p = args.Get(1).(wsfsFileInfo)
	return args.Error(0)
}

func TestFilerWorkspaceFilesExtensionsErrorsOnDupName(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		language             workspace.Language
		notebookExportFormat workspace.ExportFormat
		notebookPath         string
		filePath             string
		expectedError        string
	}{
		{
			name:                 "python source notebook and file",
			language:             workspace.LanguagePython,
			notebookExportFormat: workspace.ExportFormatSource,
			notebookPath:         "/dir/foo",
			filePath:             "/dir/foo.py",
			expectedError:        "failed to read files from the workspace file system. Duplicate paths encountered. Both NOTEBOOK at /dir/foo and FILE at /dir/foo.py resolve to the same name /foo.py. Changing the name of one of these objects will resolve this issue",
		},
		{
			name:                 "python jupyter notebook and file",
			language:             workspace.LanguagePython,
			notebookExportFormat: workspace.ExportFormatJupyter,
			notebookPath:         "/dir/foo",
			filePath:             "/dir/foo.py",
			// Jupyter notebooks would correspond to foo.ipynb so an error is not expected.
			expectedError: "",
		},
		{
			name:                 "scala source notebook and file",
			language:             workspace.LanguageScala,
			notebookExportFormat: workspace.ExportFormatSource,
			notebookPath:         "/dir/foo",
			filePath:             "/dir/foo.scala",
			expectedError:        "failed to read files from the workspace file system. Duplicate paths encountered. Both NOTEBOOK at /dir/foo and FILE at /dir/foo.scala resolve to the same name /foo.scala. Changing the name of one of these objects will resolve this issue",
		},
		{
			name:                 "r source notebook and file",
			language:             workspace.LanguageR,
			notebookExportFormat: workspace.ExportFormatSource,
			notebookPath:         "/dir/foo",
			filePath:             "/dir/foo.r",
			expectedError:        "failed to read files from the workspace file system. Duplicate paths encountered. Both NOTEBOOK at /dir/foo and FILE at /dir/foo.r resolve to the same name /foo.r. Changing the name of one of these objects will resolve this issue",
		},
		{
			name:                 "sql source notebook and file",
			language:             workspace.LanguageSql,
			notebookExportFormat: workspace.ExportFormatSource,
			notebookPath:         "/dir/foo",
			filePath:             "/dir/foo.sql",
			expectedError:        "failed to read files from the workspace file system. Duplicate paths encountered. Both NOTEBOOK at /dir/foo and FILE at /dir/foo.sql resolve to the same name /foo.sql. Changing the name of one of these objects will resolve this issue",
		},
		{
			name:                 "python jupyter notebook and file",
			language:             workspace.LanguagePython,
			notebookExportFormat: workspace.ExportFormatJupyter,
			notebookPath:         "/dir/foo",
			filePath:             "/dir/foo.ipynb",
			expectedError:        "failed to read files from the workspace file system. Duplicate paths encountered. Both NOTEBOOK at /dir/foo and FILE at /dir/foo.ipynb resolve to the same name /foo.ipynb. Changing the name of one of these objects will resolve this issue",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockedWorkspaceClient := mocks.NewMockWorkspaceClient(t)
			mockedApiClient := mockApiClient{}

			// Mock the workspace API's ListAll method.
			workspaceApi := mockedWorkspaceClient.GetMockWorkspaceAPI()
			workspaceApi.EXPECT().ListAll(mock.Anything, workspace.ListWorkspaceRequest{
				Path: "/dir",
			}).Return([]workspace.ObjectInfo{
				{
					Path:       tc.filePath,
					Language:   tc.language,
					ObjectType: workspace.ObjectTypeFile,
				},
				{
					Path:       tc.notebookPath,
					Language:   tc.language,
					ObjectType: workspace.ObjectTypeNotebook,
				},
			}, nil)

			// Mock bespoke API calls to /api/2.0/workspace/get-status, that are
			// used to figure out the right file extension for the notebook.
			statNotebook := wsfsFileInfo{
				ObjectInfo: workspace.ObjectInfo{
					Path:       tc.notebookPath,
					Language:   tc.language,
					ObjectType: workspace.ObjectTypeNotebook,
				},
				ReposExportFormat: tc.notebookExportFormat,
			}

			mockedApiClient.On("Do", mock.Anything, http.MethodGet, "/api/2.0/workspace/get-status", map[string]string(nil), map[string]string{
				"path":               tc.notebookPath,
				"return_export_info": "true",
			}, mock.AnythingOfType("*filer.wsfsFileInfo"), []func(*http.Request) error(nil)).Return(nil, statNotebook)

			workspaceFilesClient := workspaceFilesClient{
				workspaceClient: mockedWorkspaceClient.WorkspaceClient,
				apiClient:       &mockedApiClient,
				root:            NewWorkspaceRootPath("/dir"),
			}

			workspaceFilesExtensionsClient := workspaceFilesExtensionsClient{
				workspaceClient: mockedWorkspaceClient.WorkspaceClient,
				wsfs:            &workspaceFilesClient,
			}

			_, err := workspaceFilesExtensionsClient.ReadDir(context.Background(), "/")

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorAs(t, err, &duplicatePathError{})
				assert.EqualError(t, err, tc.expectedError)
			}

			// assert the mocked methods were actually called, as a sanity check.
			workspaceApi.AssertNumberOfCalls(t, "ListAll", 1)
			mockedApiClient.AssertNumberOfCalls(t, "Do", 1)
		})
	}
}
