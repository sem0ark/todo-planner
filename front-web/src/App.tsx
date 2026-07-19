import { useAuthStore } from './store/authStore';
import { Route, Switch, Link, Router } from 'wouter';
import { useHashLocation } from 'wouter/use-hash-location';
import LoginForm from './components/LoginForm';
import TokenDisplay from './components/TokenDisplay';
import CategoryList from './components/CategoryList';
import TemplateList from './components/TemplateList';
import TemplateEditor from './components/TemplateEditor';
import WeeklySchedule from './components/WeeklySchedule';
import ScheduleOverrides from './components/ScheduleOverrides';

function App() {
  const { token, clearAuth } = useAuthStore();
  const [location, setLocation] = useHashLocation();

  if (!token) {
    return (
      <div className="min-h-screen flex items-center justify-center p-6 bg-navy">
        <LoginForm />
      </div>
    );
  }

  const handleEditTemplate = (id: number) => {
    setLocation(`/templates/edit/${id}`);
  };

  const handleCreateTemplate = () => {
    setLocation('/templates/new');
  };

  const handleCloseEditor = () => {
    setLocation('/templates');
  };

  return (
    <Router hook={useHashLocation}>
      <div className="min-h-screen bg-navy">
        <nav className="border-b border-slate-grey bg-navy/80 backdrop-blur">
          <div className="max-w-6xl mx-auto px-6 py-4">
            <div className="flex items-center gap-6">
              <h1 className="text-xl font-semibold text-snow">Todo Planner</h1>
              <div className="flex gap-4">
                <Link href="/">
                  <a className={`px-3 py-1 text-sm transition-colors duration-micro ${
                    location === '/' ? 'text-snow font-semibold' : 'text-cloud hover:text-snow'
                  }`}>
                    Home
                  </a>
                </Link>
                <Link href="/categories">
                  <a className={`px-3 py-1 text-sm transition-colors duration-micro ${
                    location === '/categories' ? 'text-snow font-semibold' : 'text-cloud hover:text-snow'
                  }`}>
                    Categories
                  </a>
                </Link>
                <Link href="/templates">
                  <a className={`px-3 py-1 text-sm transition-colors duration-micro ${
                    location.startsWith('/templates') ? 'text-snow font-semibold' : 'text-cloud hover:text-snow'
                  }`}>
                    Templates
                  </a>
                </Link>
                <Link href="/schedule">
                  <a className={`px-3 py-1 text-sm transition-colors duration-micro ${
                    location === '/schedule' ? 'text-snow font-semibold' : 'text-cloud hover:text-snow'
                  }`}>
                    Schedule
                  </a>
                </Link>
              </div>
              <button
                onClick={clearAuth}
                className="ml-auto px-3 py-1 text-sm text-cloud hover:text-snow transition-colors duration-micro"
              >
                Logout
              </button>
            </div>
          </div>
        </nav>

        <main className="max-w-6xl mx-auto px-6 py-8">
          <Switch>
            <Route path="/" component={TokenDisplay} />
            <Route path="/categories" component={CategoryList} />
            <Route path="/templates">
              <TemplateList onEdit={handleEditTemplate} onCreate={handleCreateTemplate} />
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
