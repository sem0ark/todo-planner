import { useAuthStore } from "./store/authStore";
import { Route, Switch, Link, Router } from "wouter";
import { useHashLocation } from "wouter/use-hash-location";
import LoginForm from "./components/LoginForm";
import Home from "./components/Home";
import TokenDisplay from "./components/TokenDisplay";
import CategoryList from "./components/CategoryList";
import TemplateList from "./components/TemplateList";
import TemplateEditor from "./components/TemplateEditor";
import WeeklySchedule from "./components/WeeklySchedule";
import ScheduleOverrides from "./components/ScheduleOverrides";

function App() {
  const { token, clearAuth } = useAuthStore();
  const [location, setLocation] = useHashLocation();

  if (!token) {
    return (
      <div className="min-h-screen flex items-center justify-center p-6 bg-app-void">
        <LoginForm />
      </div>
    );
  }

  const handleEditTemplate = (id: number) => {
    setLocation(`/templates/edit/${id}`);
  };

  const handleCreateTemplate = () => {
    setLocation("/templates/new");
  };

  const handleCloseEditor = () => {
    setLocation("/templates");
  };

  return (
    <Router hook={useHashLocation}>
      <div className="min-h-screen bg-app-void">
        <nav className="border-b border-slate-grey/20 bg-app-void backdrop-blur">
          <div className="max-w-6xl mx-auto px-6 py-4">
            <div className="flex items-center gap-6">
              <h1 className="text-lg font-semibold text-snow">Todo Planner</h1>
              <div className="flex gap-4">
                <Link
                  href="/"
                  className={`px-3 py-1 text-sm transition-colors duration-micro ${
                    location === "/"
                      ? "text-snow font-semibold"
                      : "text-slate-blue hover:text-cloud"
                  }`}
                >
                  Home
                </Link>
                <Link
                  href="/review"
                  className={`px-3 py-1 text-sm transition-colors duration-micro ${
                    location === "/review"
                      ? "text-snow font-semibold"
                      : "text-slate-blue hover:text-cloud"
                  }`}
                >
                  Review
                </Link>
                <Link
                  href="/categories"
                  className={`px-3 py-1 text-sm transition-colors duration-micro ${
                    location === "/categories"
                      ? "text-snow font-semibold"
                      : "text-slate-blue hover:text-cloud"
                  }`}
                >
                  Categories
                </Link>
                <Link
                  href="/templates"
                  className={`px-3 py-1 text-sm transition-colors duration-micro ${
                    location.startsWith("/templates")
                      ? "text-snow font-semibold"
                      : "text-slate-blue hover:text-cloud"
                  }`}
                >
                  Templates
                </Link>
                <Link
                  href="/schedule"
                  className={`px-3 py-1 text-sm transition-colors duration-micro ${
                    location === "/schedule"
                      ? "text-snow font-semibold"
                      : "text-slate-blue hover:text-cloud"
                  }`}
                >
                  Schedule
                </Link>
              </div>
              <button
                onClick={clearAuth}
                className="ml-auto px-3 py-1 text-sm text-slate-blue hover:text-cloud transition-colors duration-micro"
              >
                Logout
              </button>
            </div>
          </div>
        </nav>

        <main className="max-w-6xl mx-auto px-6 py-8">
          <Switch>
            <Route path="/" component={Home} />
            <Route path="/token" component={TokenDisplay} />
            <Route path="/categories" component={CategoryList} />
            <Route path="/review">
              <div className="text-snow">Review is coming soon.</div>
            </Route>
            <Route path="/templates">
              <TemplateList
                onEdit={handleEditTemplate}
                onCreate={handleCreateTemplate}
              />
            </Route>
            <Route path="/templates/new">
              <TemplateEditor templateId={null} onClose={handleCloseEditor} />
            </Route>
            <Route path="/templates/edit/:id">
              {(params) => (
                <TemplateEditor
                  templateId={params.id ? parseInt(params.id) : null}
                  onClose={handleCloseEditor}
                />
              )}
            </Route>
            <Route path="/schedule">
              <div className="space-y-8">
                <WeeklySchedule />
                <ScheduleOverrides />
              </div>
            </Route>
            <Route>
              <div className="text-snow">404 - Not Found</div>
            </Route>
          </Switch>
        </main>
      </div>
    </Router>
  );
}

export default App;
