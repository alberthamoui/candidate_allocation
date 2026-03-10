import { motion } from "framer-motion";
import { CheckCircleIcon, ArrowPathIcon } from "@heroicons/react/24/outline";
import { useNavigate } from "react-router-dom";

export default function SuccessPage() {
	const navigate = useNavigate();

	return (
		<div className="min-h-screen bg-gradient-to-br from-green-50 via-emerald-50 to-teal-50 flex items-center justify-center p-6">
			<motion.div
				initial={{ opacity: 0, scale: 0.9 }}
				animate={{ opacity: 1, scale: 1 }}
				transition={{ duration: 0.5 }}
				className="max-w-md w-full bg-white rounded-3xl shadow-2xl p-10 text-center"
			>
				<motion.div
					initial={{ scale: 0 }}
					animate={{ scale: 1 }}
					transition={{ delay: 0.2, type: "spring", stiffness: 200 }}
					className="w-24 h-24 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-8"
				>
					<CheckCircleIcon className="w-14 h-14 text-green-600" />
				</motion.div>

				<h1 className="text-3xl font-bold text-gray-900 mb-4">
					Tudo Pronto!
				</h1>
				<p className="text-gray-600 mb-10 leading-relaxed">
					Os candidatos e restrições foram salvos com sucesso no banco
					de dados. O sistema está pronto para a próxima etapa.
				</p>

				<div className="space-y-4">
					<motion.button
						whileHover={{ scale: 1.02 }}
						whileTap={{ scale: 0.98 }}
						onClick={() => navigate("/")}
						className="w-full flex items-center justify-center space-x-2 bg-gradient-to-r from-green-600 to-emerald-700 text-white py-4 rounded-xl font-bold shadow-lg hover:shadow-xl transition-all"
					>
						<ArrowPathIcon className="w-5 h-5" />
						<span>Nova Importação</span>
					</motion.button>
				</div>
			</motion.div>
		</div>
	);
}
