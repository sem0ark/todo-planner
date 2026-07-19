import { Link } from 'wouter';
import { useAuthStore } from '../store/authStore';

export default function Home() {
  const { user } = useAuthStore();

  return (
    <div className="w-full max-w-4xl mx-auto">
      <div className="text-center mb-12">
        <h1 className="text-4xl font-bold text-snow mb-4">
          Welcome to Todo Planner
        </h1>
        <p className="text-lg text-cloud">
          {user ? `Hello, ${user.username}!` : 'Plan your day with ease'}
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-12">
        <Link href="/categories">
          <div className="p-8 bg-slate-blue/10 border border-slate-grey rounded-lg hover:bg-slate-blue/20 transition-colors cursor-pointer">
            <div className="text-4xl mb-4">📋</div>
            <h3 className="text-xl font-semibold text-snow mb-2">Categories</h3>
            <p className="text-cloud text-sm">
              Create and manage task categories with custom colors
            </p>
          </div>
        </Link>

        <Link href="/templates">
          <div className="p-8 bg-slate-blue/10 border border-slate-grey rounded-lg hover:bg-slate-blue/20 transition-colors cursor-pointer">
            <div className="text-4xl mb-4">📝</div>
            <h3 className="text-xl font-semibold text-snow mb-2">Templates</h3>
            <p className="text-cloud text-sm">
              Design daily schedule templates with drag-and-drop blocks
            </p>
          </div>
        </Link>

        <Link href="/schedule">
          <div className="p-8 bg-slate-blue/10 border border-slate-grey rounded-lg hover:bg-slate-blue/20 transition-colors cursor-pointer">
            <div className="text-4xl mb-4">📅</div>
            <h3 className="text-xl font-semibold text-snow mb-2">Schedule</h3>
            <p className="text-cloud text-sm">
              View and manage your weekly schedule
            </p>
          </div>
        </Link>

        <Link href="/token">
          <div className="p-8 bg-slate-blue/10 border border-slate-grey rounded-lg hover:bg-slate-blue/20 transition-colors cursor-pointer">
            <div className="text-4xl mb-4">🖥️</div>
            <h3 className="text-xl font-semibold text-snow mb-2">Desktop Widget</h3>
            <p className="text-cloud text-sm">
              Open the macOS desktop widget application
            </p>
          </div>
        </Link>
      </div>

      <div className="text-center">
        <Link href="/token">
          <span className="text-sm text-cloud hover:text-snow transition-colors underline cursor-pointer">
            View authentication token
          </span>
        </Link>
      </div>
    </div>
  );
}
